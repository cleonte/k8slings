package exercises

import "fmt"

// SkipReason says why the exercise cannot run on this machine, or is empty if
// it can. Only manifest exercises run today.
func (e Exercise) SkipReason() string {
	if e.Mode != ModeManifest {
		return fmt.Sprintf("%s exercises are not available yet", e.Mode)
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
func Skipped(infoFile string) ([]Exercise, error) {
	all, err := List(infoFile)
	if err != nil {
		return nil, err
	}
	var skipped []Exercise
	for _, e := range all {
		if e.SkipReason() != "" {
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
