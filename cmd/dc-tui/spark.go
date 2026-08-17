package main

const sparkCap = 60

var sparkBlocks = []rune("▁▂▃▄▅▆▇█")

type sparkHist struct {
	cpu []float64
	mem []float64
}

func appendSpark(h sparkHist, cpu, mem float64) sparkHist {
	h.cpu = append(h.cpu, cpu)
	h.mem = append(h.mem, mem)
	if len(h.cpu) > sparkCap {
		h.cpu = append([]float64(nil), h.cpu[len(h.cpu)-sparkCap:]...)
		h.mem = append([]float64(nil), h.mem[len(h.mem)-sparkCap:]...)
	}
	return h
}

func sparkline(vals []float64, width int) string {
	if width < 2 {
		return ""
	}
	if len(vals) == 0 {
		return ""
	}
	n := width
	if n > len(vals) {
		n = len(vals)
	}
	slice := vals[len(vals)-n:]
	max := 0.0
	for _, v := range slice {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		max = 1
	}
	out := make([]rune, len(slice))
	last := len(sparkBlocks) - 1
	for i, v := range slice {
		if v < 0 {
			v = 0
		}
		idx := int((v / max) * float64(last))
		if idx > last {
			idx = last
		}
		out[i] = sparkBlocks[idx]
	}
	return string(out)
}
