package exercises

import "fmt"

// Availability is what this machine can run: whether the k8slings cluster is
// there and healthy, and how many nodes it has.
type Availability struct {
	Cluster bool
	Nodes   int
}

// SkipReason says why the exercise cannot run on this machine, or is empty if
// it can. Manifest exercises always run.
func (e Exercise) SkipReason(av Availability) string {
	switch e.Mode {
	case ModeFix:
		return "fix exercises are not available yet"
	case ModeCluster:
		if !av.Cluster {
			return "needs the cluster: run `k8slings cluster up`"
		}
	}
	for _, need := range e.Requires {
		switch need {
		case "multinode":
			if av.Nodes < 3 {
				return "needs several nodes: run `k8slings cluster down`, then `k8slings cluster up` without --single"
			}
		default:
			return fmt.Sprintf("needs the %s add-on, which is not available yet", need)
		}
	}
	return ""
}

// Progress counts the exercises that are done and the exercises there are.
func Progress(infoFile string) (done, total int, err error) {
	all, err := List(infoFile)
	if err != nil {
		return 0, 0, err
	}
	for _, e := range all {
		if e.State() == Done {
			done++
		}
	}
	return done, len(all), nil
}

// Skipped returns the exercises that cannot run here.
func Skipped(infoFile string, av Availability) ([]Exercise, error) {
	all, err := List(infoFile)
	if err != nil {
		return nil, err
	}
	var skipped []Exercise
	for _, e := range all {
		if e.SkipReason(av) != "" {
			skipped = append(skipped, e)
		}
	}
	return skipped, nil
}

// TopicProgress is how far the learner is in one topic.
type TopicProgress struct {
	Topic       string
	Done, Total int
}

// ProgressByTopic returns the progress of every topic, in manifest order.
func ProgressByTopic(infoFile string) ([]TopicProgress, error) {
	all, err := List(infoFile)
	if err != nil {
		return nil, err
	}
	var out []TopicProgress
	for _, e := range all {
		if len(out) == 0 || out[len(out)-1].Topic != e.Topic {
			out = append(out, TopicProgress{Topic: e.Topic})
		}
		last := &out[len(out)-1]
		last.Total++
		if e.State() == Done {
			last.Done++
		}
	}
	return out, nil
}
