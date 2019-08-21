package stomvu

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type PlanTarget struct {
	Target  string
	Sources []string
}

type Plan struct {
	FileTargets      []*PlanTarget
	DirectoryTargets []*PlanTarget
	Dunno            []string // includes both files and folders
}

func (p *Plan) InsertFile(source string, targetDir string) {
	for _, target := range p.FileTargets {
		if target.Target == targetDir {
			target.Sources = append(target.Sources, source)
			return
		}
	}

	// no match => add new target
	p.FileTargets = append(p.FileTargets, &PlanTarget{
		Target:  targetDir,
		Sources: []string{source},
	})
}

func computePlan(targetFn func(string) string) (*Plan, error) {
	plan := Plan{
		FileTargets:      []*PlanTarget{},
		DirectoryTargets: []*PlanTarget{},
		Dunno:            []string{},
	}

	dentries, err := ioutil.ReadDir(".")
	if err != nil {
		return nil, err
	}

	for _, dentry := range dentries {
		target := targetFn(dentry.Name())
		if target == "" {
			maybeDirSuffix := ""
			if dentry.IsDir() {
				maybeDirSuffix = "/"
			}

			plan.Dunno = append(plan.Dunno, dentry.Name()+maybeDirSuffix)
			continue
		}

		if dentry.IsDir() {
			if dentry.Name() == target { // already right name
				continue
			}

			plan.DirectoryTargets = append(plan.DirectoryTargets, &PlanTarget{
				Target:  target,
				Sources: []string{dentry.Name()},
			})
		} else {
			plan.InsertFile(dentry.Name(), target)
		}
	}

	return &plan, nil
}

func executePlan(plan *Plan) error {
	for _, target := range plan.FileTargets {
		for _, source := range target.Sources {
			if err := os.MkdirAll(target.Target, 0755); err != nil {
				return err
			}

			if err := os.Rename(source, target.Target+"/"+source); err != nil {
				return err
			}
		}
	}

	for _, target := range plan.DirectoryTargets {
		for _, source := range target.Sources {
			if err := os.Rename(source, target.Target); err != nil {
				return err
			}
		}
	}

	return nil
}

func explainPlan(plan *Plan, out io.Writer) {
	for _, target := range plan.FileTargets {
		fmt.Fprintf(out, "%s <= %v\n", target.Target, target.Sources)
	}

	for _, target := range plan.DirectoryTargets {
		fmt.Fprintf(out, "%s/ <= %v/\n", target.Target, target.Sources)
	}

	if len(plan.Dunno) > 0 {
		fmt.Fprintf(out, "\nDUNNO\n-------\n")

		for _, dunno := range plan.Dunno {
			fmt.Fprintf(out, "%s\n", dunno)
		}
	}
}
