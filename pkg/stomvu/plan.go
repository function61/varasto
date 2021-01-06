// "Move utils" - logic for moving files to hierarchies sensible for TV shows, photo albums etc.
package stomvu

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

type PlanTarget struct {
	Target  string
	Sources []string
}

type Plan struct {
	Dir              string // directory in which both FileTargets and DirectoryTargets reside
	FileTargets      []*PlanTarget
	DirectoryTargets []*PlanTarget
	Dunno            []string // includes both files and folders
}

// total number of sources in all plan targets (= num of files and directories to move)
func (p *Plan) NumSources() int {
	numSources := func(over []*PlanTarget) int {
		num := 0

		for _, target := range over {
			num += len(target.Sources)
		}

		return num
	}

	return numSources(p.FileTargets) + numSources(p.DirectoryTargets)
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

func ComputePlan(dir string, targetFn func(string) string) (*Plan, error) {
	plan := Plan{
		Dir:              dir,
		FileTargets:      []*PlanTarget{},
		DirectoryTargets: []*PlanTarget{},
		Dunno:            []string{},
	}

	dentries, err := ioutil.ReadDir(dir)
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

	sort.Slice(plan.FileTargets, func(i, j int) bool {
		return plan.FileTargets[i].Target < plan.FileTargets[j].Target
	})

	return &plan, nil
}

// does what plan suggested to do
func ExecutePlan(plan *Plan) error {
	// files which to move into other directories
	for _, target := range plan.FileTargets {
		for _, source := range target.Sources {
			targetDir := filepath.Join(plan.Dir, target.Target)

			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return err
			}

			if err := os.Rename(filepath.Join(plan.Dir, source), filepath.Join(targetDir, source)); err != nil {
				return err
			}
		}
	}

	// directories which to rename
	for _, target := range plan.DirectoryTargets {
		for _, source := range target.Sources {
			if err := os.Rename(filepath.Join(plan.Dir, source), filepath.Join(plan.Dir, target.Target)); err != nil {
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
