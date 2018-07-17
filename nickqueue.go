package main

const Q_LEN = 5

// NOTE(tso): alternatively, use make([]string, capacity)
//            func NewNickQueue(capacity int) *nickQueue { // ...
//            for i := cap(q) - 1; // ... in Push()

type nickQueue [Q_LEN]string

func (q *nickQueue) Push(n string) {
	for i := Q_LEN - 1; i != 0; i-- {
		q[i] = q[i-1]
	}
	q[0] = n
}

func (q *nickQueue) Max() int {
	max := 0
	for _, n := range q {
		if len(n) > max {
			max = len(n)
		}
	}
	return max
}

func (q *nickQueue) Mode() int {
	m := map[int]int{}
	for _, n := range q {
		m[len(n)]++
	}
	max, mode := 0, 0
	for l, c := range m {
		if l > 0 && c > max {
			max = c
			mode = l
		}
	}
	return mode
}
