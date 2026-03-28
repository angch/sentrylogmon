#!/bin/bash
echo "Looking for UX improvements..."
echo ""
echo "Files with 'TODO':"
grep -r "TODO" . | grep -v "vendor" | head -n 10
echo ""
echo "Files with 'FIXME':"
grep -r "FIXME" . | grep -v "vendor" | head -n 10
