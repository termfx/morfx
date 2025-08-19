package util

// RemapAllMatches remapea todos los índices (match + submatches) desde la cadena
// normalizada (bytes) a la original (bytes). Si algún índice no puede remapearse
// exactamente, los ajusta buscando el byte mapeable más cercano hacia la derecha
// (para starts) o hacia la izquierda (para ends).
func RemapAllMatches(matches [][]int, n2o, o2n []int) [][]int {
	out := make([][]int, len(matches))
	for i, m := range matches {
		if m == nil {
			continue
		}
		rm := make([]int, len(m))
		for j := 0; j < len(m); j += 2 {
			ns, ne := m[j], m[j+1]
			if ns < 0 || ne < 0 {
				rm[j], rm[j+1] = -1, -1
				continue
			}
			os := mapStartIdx(ns, n2o)     // busca hacia la derecha si cae en zona no mapeada
			oe := mapEndIdx(ne-1, n2o) + 1 // busca hacia la izquierda si cae en zona no mapeada
			if os < 0 || oe < 0 || os > oe {
				rm[j], rm[j+1] = -1, -1
			} else {
				rm[j], rm[j+1] = os, oe
			}
		}
		out[i] = rm
	}
	return out
}

// mapStartIdx: dado un índice byte en la normalizada, devuelve el primer byte original
// mapeable en n2o empezando en idx y yendo a la derecha.
func mapStartIdx(idx int, n2o []int) int {
	for i := idx; i < len(n2o); i++ {
		if n2o[i] >= 0 {
			return n2o[i]
		}
	}
	// Si idx == len(normalized) (match al final), no hay byte propio -> devolvemos último mapeable +1
	if idx == len(n2o) && len(n2o) > 0 {
		last := n2o[len(n2o)-1]
		if last >= 0 {
			return last + 1
		}
	}
	return -1
}

// mapEndIdx: dado un índice byte (inclusive) en la normalizada, devuelve el último
// byte original mapeable buscando hacia la izquierda.
func mapEndIdx(idx int, n2o []int) int {
	if idx >= len(n2o) {
		idx = len(n2o) - 1
	}
	for i := idx; i >= 0; i-- {
		if n2o[i] >= 0 {
			return n2o[i]
		}
	}
	return -1
}
