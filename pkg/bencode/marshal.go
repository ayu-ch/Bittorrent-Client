package bencode

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
)

func Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := marshalValue(v, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func marshalValue(v any, b *bytes.Buffer) error {
	switch value := v.(type) {
	case int:
		marshalInt(value, b)

	case string:
		marshalString(value, b)
	case []any:
		marshalList(value, b)
	case map[string]any:
		marshalDict(value, b)
	default:
		return fmt.Errorf("Unsupported type:%T", v)
	}
	return nil
}

func marshalInt(v int, b *bytes.Buffer) {
	b.WriteRune('i')
	b.WriteString(strconv.Itoa(v))
	b.WriteRune('e')
}

func marshalString(s string, b *bytes.Buffer) {
	b.WriteString(strconv.Itoa(len(s)))
	b.WriteRune(':')
	b.WriteString(s)
}

func marshalList(list []any, b *bytes.Buffer) {
	b.WriteRune('l')
	for _, item := range list {
		if err := marshalValue(item, b); err != nil {
			return
		}
	}
	b.WriteRune('e')
}

func marshalDict(dict map[string]any, buf *bytes.Buffer) {
	buf.WriteRune('d')
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	// Sort keys for consistent output
	sort.Strings(keys)
	for _, k := range keys {
		marshalString(k, buf)
		if err := marshalValue(dict[k], buf); err != nil {
			return
		}
	}
	buf.WriteRune('e')
}
