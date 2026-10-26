#!/usr/bin/env python3
"""Emit a Go repeat driver for generated app.Main(); put it inside that module."""
import argparse
import json
from pathlib import Path
p = argparse.ArgumentParser()
p.add_argument('import_path')
p.add_argument('output', type=Path)
a = p.parse_args()
a.output.parent.mkdir(parents=True, exist_ok=True)
a.output.write_text('''package main
import (
    "fmt"
    "os"
    "strconv"
    "time"
    app %s
)
func main() {
    if len(os.Args) != 3 { panic("expected warmups samples") }
    warmups, err := strconv.Atoi(os.Args[1]); if err != nil { panic(err) }
    samples, err := strconv.Atoi(os.Args[2]); if err != nil { panic(err) }
    for index := -warmups; index < samples; index++ {
        start := time.Now()
        app.Main()
        elapsed := time.Since(start).Nanoseconds()
        fmt.Fprintf(os.Stderr, "PERF\\t%%d\\t%%d\\n", index, elapsed)
    }
}
''' % json.dumps(a.import_path))
