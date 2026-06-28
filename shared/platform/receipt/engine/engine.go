package engine

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"mh-pos-platform/receipt/ir"
	"mh-pos-platform/receipt/parser"
)

var templateExprRE = regexp.MustCompile(`\{\{\s*\.([A-Za-z0-9_]+)\s*(?:\|\s*([A-Za-z0-9_]+)\s*)?\}\}`)

// Render раскрывает ReceiptLine Level 1 template поверх print context и
// возвращает IR без управляющих IfBlock/EachBlock. Функция детерминирована:
// она не читает внешнее состояние, БД, конфиги принтера или текущее время.
func Render(templateContent string, printContext any) ([]ir.Block, error) {
	blocks, err := parser.Parse(templateContent)
	if err != nil {
		return nil, err
	}
	root := scopeFromValue(printContext)
	return renderBlocks(blocks, []scope{root})
}

type scope map[string]any

func renderBlocks(blocks []ir.Block, scopes []scope) ([]ir.Block, error) {
	out := make([]ir.Block, 0, len(blocks))
	for _, block := range blocks {
		rendered, err := renderBlock(block, scopes)
		if err != nil {
			return nil, err
		}
		out = append(out, rendered...)
	}
	return out, nil
}

func renderBlock(block ir.Block, scopes []scope) ([]ir.Block, error) {
	switch v := block.(type) {
	case ir.TextBlock:
		block := cloneTextBlock(v)
		for i := range block.Lines {
			for j := range block.Lines[i].Columns {
				text, err := renderString(block.Lines[i].Columns[j].Text, scopes)
				if err != nil {
					return nil, err
				}
				block.Lines[i].Columns[j].Text = text
			}
		}
		return []ir.Block{block}, nil
	case ir.QRBlock:
		payload, err := renderString(v.Payload, scopes)
		if err != nil {
			return nil, err
		}
		v.Payload = payload
		return []ir.Block{v}, nil
	case ir.BarcodeBlock:
		data, err := renderString(v.Data, scopes)
		if err != nil {
			return nil, err
		}
		v.Data = data
		return []ir.Block{v}, nil
	case ir.IfBlock:
		if !truthy(lookup(scopes, v.Expr)) {
			return nil, nil
		}
		return renderBlocks(v.Blocks, scopes)
	case ir.EachBlock:
		values := iterable(lookup(scopes, v.Key))
		out := make([]ir.Block, 0, len(values)*len(v.Blocks))
		for _, value := range values {
			child := append(append([]scope(nil), scopes...), scopeFromValue(value))
			rendered, err := renderBlocks(v.Blocks, child)
			if err != nil {
				return nil, err
			}
			out = append(out, rendered...)
		}
		return out, nil
	default:
		return []ir.Block{block}, nil
	}
}

func cloneTextBlock(block ir.TextBlock) ir.TextBlock {
	out := block
	out.Lines = make([]ir.TextLine, len(block.Lines))
	for i := range block.Lines {
		out.Lines[i].Columns = append([]ir.Column(nil), block.Lines[i].Columns...)
	}
	return out
}

func renderString(input string, scopes []scope) (string, error) {
	var renderErr error
	out := templateExprRE.ReplaceAllStringFunc(input, func(match string) string {
		if renderErr != nil {
			return ""
		}
		parts := templateExprRE.FindStringSubmatch(match)
		if len(parts) < 3 {
			renderErr = fmt.Errorf("invalid template expression %q", match)
			return ""
		}
		value := lookup(scopes, parts[1])
		if parts[2] == "money" {
			minor, ok := asInt64(value)
			if !ok {
				renderErr = fmt.Errorf("money pipe expects integer minor units for %q", parts[1])
				return ""
			}
			return FormatMoneyMinor(minor, currencyCode(scopes))
		}
		if parts[2] != "" {
			renderErr = fmt.Errorf("unsupported template pipe %q", parts[2])
			return ""
		}
		return stringify(value)
	})
	if renderErr != nil {
		return "", renderErr
	}
	return out, nil
}

func lookup(scopes []scope, key string) any {
	key = strings.TrimSpace(key)
	for i := len(scopes) - 1; i >= 0; i-- {
		if value, ok := scopes[i][key]; ok {
			return value
		}
	}
	return nil
}

func currencyCode(scopes []scope) string {
	if value, ok := lookup(scopes, "currency_code").(string); ok {
		return value
	}
	return ""
}

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int8, int16, int32, int64:
		return strconv.FormatInt(reflect.ValueOf(v).Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(reflect.ValueOf(v).Uint(), 10)
	case float32, float64:
		return strconv.FormatFloat(reflect.ValueOf(v).Float(), 'f', -1, 64)
	default:
		return fmt.Sprint(v)
	}
}

func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return strings.TrimSpace(v) != ""
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int() != 0
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint() != 0
	case float32, float64:
		return reflect.ValueOf(v).Float() != 0
	case []any:
		return len(v) != 0
	case scope:
		return len(v) != 0
	default:
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Pointer, reflect.Interface:
			if rv.IsNil() {
				return false
			}
			return truthy(rv.Elem().Interface())
		case reflect.Slice, reflect.Array, reflect.Map:
			return rv.Len() != 0
		}
		return true
	}
}

func iterable(value any) []any {
	if value == nil {
		return nil
	}
	if items, ok := value.([]any); ok {
		return items
	}
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil
	}
	out := make([]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out = append(out, rv.Index(i).Interface())
	}
	return out
}

func asInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8, int16, int32, int64:
		return reflect.ValueOf(v).Int(), true
	case uint, uint8, uint16, uint32, uint64:
		n := reflect.ValueOf(v).Uint()
		if n > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(n), true
	case float32, float64:
		f := reflect.ValueOf(v).Float()
		if f != float64(int64(f)) {
			return 0, false
		}
		return int64(f), true
	default:
		return 0, false
	}
}

func scopeFromValue(value any) scope {
	out := scope{}
	addValueToScope(out, reflect.ValueOf(value))
	return out
}

func addValueToScope(out scope, rv reflect.Value) {
	if !rv.IsValid() {
		return
	}
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Struct:
		rt := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			field := rt.Field(i)
			if field.PkgPath != "" {
				continue
			}
			value := normalizeValue(rv.Field(i))
			for _, key := range fieldKeys(field) {
				out[key] = value
			}
		}
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return
		}
		iter := rv.MapRange()
		for iter.Next() {
			out[iter.Key().String()] = normalizeValue(iter.Value())
		}
	default:
		out["value"] = normalizeValue(rv)
	}
}

func fieldKeys(field reflect.StructField) []string {
	keys := []string{field.Name}
	if jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]; jsonTag != "" && jsonTag != "-" {
		keys = append(keys, jsonTag)
	}
	return keys
}

func normalizeValue(rv reflect.Value) any {
	if !rv.IsValid() {
		return nil
	}
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Struct:
		out := scope{}
		addValueToScope(out, rv)
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out = append(out, normalizeValue(rv.Index(i)))
		}
		return out
	case reflect.Map:
		out := scope{}
		addValueToScope(out, rv)
		return out
	default:
		return rv.Interface()
	}
}
