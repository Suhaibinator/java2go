// javacampaign independently reruns one frozen-seed differential fixture.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/NickyBoy89/java2go/campaign"
	"os"
	"time"
)

func main() {
	var c campaign.Config
	flag.StringVar(&c.Repository, "repository", ".", "java2go checkout")
	flag.StringVar(&c.Fixture, "fixture", "", "fixture directory containing fixture.json")
	flag.StringVar(&c.Artifacts, "artifacts", "", "preserved run artifact parent (default .campaign/runs)")
	flag.StringVar(&c.JDK, "jdk", campaign.DefaultJDK, "JDK 21 home; never ambient java/javac")
	flag.StringVar(&c.Transpiler, "transpiler", "", "prebuilt transpiler executable (default build isolated executable)")
	flag.DurationVar(&c.BuildTimeout, "build-timeout", 5*time.Minute, "maximum duration per build stage")
	flag.DurationVar(&c.RunTimeout, "run-timeout", time.Minute, "maximum duration per individual program execution")
	flag.BoolVar(&c.Race, "race", false, "build every generated Go package and application with the race detector")
	flag.IntVar(&c.StressRuns, "stress-runs", 0, "additional isolated Go executions, cycling frozen seeds against validated JVM observations")
	flag.Parse()
	if c.Fixture == "" {
		fmt.Fprintln(os.Stderr, "-fixture is required")
		os.Exit(2)
	}
	report, err := campaign.Run(context.Background(), c)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
