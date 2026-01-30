use serde::{Deserialize, Serialize};
use std::sync::Arc;
use sysinfo::System;
use tokio::sync::RwLock;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemState {
    pub load_avg: LoadAverage,
    pub memory: MemoryInfo,
    pub top_processes: Vec<ProcessInfo>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoadAverage {
    pub one: f64,
    pub five: f64,
    pub fifteen: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryInfo {
    pub total: u64,
    pub used: u64,
    pub available: u64,
    pub percent: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProcessInfo {
    pub pid: u32,
    pub name: String,
    pub cpu_percent: f32,
    pub memory_mb: u64,
}

pub struct Collector {
    state: Arc<RwLock<SystemState>>,
}

impl Collector {
    pub fn new() -> Self {
        let state = Arc::new(RwLock::new(SystemState {
            load_avg: LoadAverage {
                one: 0.0,
                five: 0.0,
                fifteen: 0.0,
            },
            memory: MemoryInfo {
                total: 0,
                used: 0,
                available: 0,
                percent: 0.0,
            },
            top_processes: Vec::new(),
        }));

        Self { state }
    }

    pub async fn run(&self) {
        let state = self.state.clone();
        tokio::spawn(async move {
            let mut sys = System::new_all();

            loop {
                // Performance optimization: Use granular refresh methods instead of
                // refresh_all() to avoid refreshing unnecessary data (disks, networks,
                // temperature sensors, etc.). We only need memory and process info.
                sys.refresh_memory();
                sys.refresh_processes();

                let load_avg = System::load_average();
                let total_mem = sys.total_memory();
                let used_mem = sys.used_memory();
                let available_mem = sys.available_memory();

                let mut processes: Vec<_> = sys.processes().values().collect();
                processes.sort_by(|a, b| {
                    b.cpu_usage()
                        .partial_cmp(&a.cpu_usage())
                        .unwrap_or(std::cmp::Ordering::Equal)
                });

                let top_processes: Vec<ProcessInfo> = processes
                    .iter()
                    .take(5)
                    .map(|p| ProcessInfo {
                        pid: p.pid().as_u32(),
                        name: p.name().to_string(),
                        cpu_percent: p.cpu_usage(),
                        memory_mb: p.memory() / 1024 / 1024,
                    })
                    .collect();

                let new_state = SystemState {
                    load_avg: LoadAverage {
                        one: load_avg.one,
                        five: load_avg.five,
                        fifteen: load_avg.fifteen,
                    },
                    memory: MemoryInfo {
                        total: total_mem,
                        used: used_mem,
                        available: available_mem,
                        percent: (used_mem as f64 / total_mem as f64) * 100.0,
                    },
                    top_processes,
                };

                *state.write().await = new_state;

                // Backoff logic: if load > num_cpus, sleep longer
                let sleep_duration = if load_avg.one > sys.cpus().len() as f64 {
                    tokio::time::Duration::from_secs(600) // 10 minutes
                } else {
                    tokio::time::Duration::from_secs(60) // 1 minute
                };

                tokio::time::sleep(sleep_duration).await;
            }
        });
    }

    #[allow(dead_code)]
    pub async fn get_state(&self) -> SystemState {
        self.state.read().await.clone()
    }
}
