package application

import taskdomain "alter0/internal/task/domain"

type recorder interface {
	Record(task taskdomain.Task)
}

type RecorderGroup struct {
	recorders []recorder
}

func NewRecorderGroup(recorders ...recorder) *RecorderGroup {
	items := make([]recorder, 0, len(recorders))
	for _, item := range recorders {
		if item == nil {
			continue
		}
		items = append(items, item)
	}
	return &RecorderGroup{recorders: items}
}

func (g *RecorderGroup) Record(task taskdomain.Task) {
	if g == nil {
		return
	}
	for _, item := range g.recorders {
		item.Record(task)
	}
}
