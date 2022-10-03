package alerter

import "github.com/zeebo/xxh3"

func xxh64FromString(s ...string) uint64 {
	d := xxh3.New()
	if len(s) == 0 {
		return 0
	}
	if len(s) == 1 {
		d.WriteString(s[0])
	} else {
		for _, v := range s {
			d.WriteString(v)
		}
	}

	return d.Sum64()
}
