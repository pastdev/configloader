package config

import "fmt"

func normalizeDocument(node any) any {
	switch typed := node.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, v := range typed {
			out[k] = normalizeDocument(v)
		}
		return out

	case map[any]any:
		out := map[string]any{}
		for k, v := range typed {
			if casted, ok := k.(string); ok {
				out[casted] = normalizeDocument(v)
			} else {
				out[fmt.Sprint(k)] = normalizeDocument(v)
			}
		}
		return out

	case []any:
		out := make([]any, len(typed))
		for i, v := range typed {
			out[i] = normalizeDocument(v)
		}
		return out

	default:
		return node
	}
}

func mergeDocument(dst any, src any) any {
	if dst == nil {
		return src
	}

	dstMap, dstOK := dst.(map[string]any)
	srcMap, srcOK := src.(map[string]any)
	if dstOK && srcOK {
		for k, v := range srcMap {
			dstMap[k] = mergeDocument(dstMap[k], v)
		}
		return dstMap
	}

	return src
}
