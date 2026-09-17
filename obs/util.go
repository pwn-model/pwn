package obs

import "github.com/mlange-42/ark/ecs"

func count(f *ecs.Filter0) int {
	q := f.Query()
	n := q.Count()
	q.Close()
	return n
}
