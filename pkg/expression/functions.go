// Package expression provides built-in functions for expressions
package expression

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// String functions

func funcUppercase(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("uppercase requires 1 argument")
	}
	return strings.ToUpper(fmt.Sprintf("%v", args[0])), nil
}

func funcLowercase(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("lowercase requires 1 argument")
	}
	return strings.ToLower(fmt.Sprintf("%v", args[0])), nil
}

func funcTrim(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("trim requires 1 argument")
	}
	return strings.TrimSpace(fmt.Sprintf("%v", args[0])), nil
}

func funcLength(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("length requires 1 argument")
	}
	switch v := args[0].(type) {
	case string:
		return len(v), nil
	case []interface{}:
		return len(v), nil
	case map[string]interface{}:
		return len(v), nil
	default:
		return len(fmt.Sprintf("%v", v)), nil
	}
}

func funcSubstring(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("substring requires at least 2 arguments")
	}
	s := fmt.Sprintf("%v", args[0])
	start := toInt(args[1])
	
	if start < 0 || start >= len(s) {
		return "", nil
	}
	
	if len(args) >= 3 {
		end := toInt(args[2])
		if end > len(s) {
			end = len(s)
		}
		return s[start:end], nil
	}
	
	return s[start:], nil
}

func funcReplace(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("replace requires 3 arguments")
	}
	s := fmt.Sprintf("%v", args[0])
	old := fmt.Sprintf("%v", args[1])
	new := fmt.Sprintf("%v", args[2])
	return strings.ReplaceAll(s, old, new), nil
}

func funcSplit(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("split requires 2 arguments")
	}
	s := fmt.Sprintf("%v", args[0])
	sep := fmt.Sprintf("%v", args[1])
	parts := strings.Split(s, sep)
	result := make([]interface{}, len(parts))
	for i, p := range parts {
		result[i] = p
	}
	return result, nil
}

func funcJoin(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("join requires 2 arguments")
	}
	arr, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("first argument must be an array")
	}
	sep := fmt.Sprintf("%v", args[1])
	parts := make([]string, len(arr))
	for i, v := range arr {
		parts[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(parts, sep), nil
}

func funcContains(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("contains requires 2 arguments")
	}
	s := fmt.Sprintf("%v", args[0])
	substr := fmt.Sprintf("%v", args[1])
	return strings.Contains(s, substr), nil
}

func funcStartsWith(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("startsWith requires 2 arguments")
	}
	s := fmt.Sprintf("%v", args[0])
	prefix := fmt.Sprintf("%v", args[1])
	return strings.HasPrefix(s, prefix), nil
}

func funcEndsWith(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("endsWith requires 2 arguments")
	}
	s := fmt.Sprintf("%v", args[0])
	suffix := fmt.Sprintf("%v", args[1])
	return strings.HasSuffix(s, suffix), nil
}

// Number functions

func funcRound(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("round requires 1 argument")
	}
	n := toFloat(args[0])
	precision := 0
	if len(args) >= 2 {
		precision = toInt(args[1])
	}
	multiplier := math.Pow(10, float64(precision))
	return math.Round(n*multiplier) / multiplier, nil
}

func funcFloor(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("floor requires 1 argument")
	}
	return math.Floor(toFloat(args[0])), nil
}

func funcCeil(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("ceil requires 1 argument")
	}
	return math.Ceil(toFloat(args[0])), nil
}

func funcAbs(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("abs requires 1 argument")
	}
	return math.Abs(toFloat(args[0])), nil
}

func funcMin(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("min requires at least 1 argument")
	}
	
	// Handle array input
	if arr, ok := args[0].([]interface{}); ok {
		if len(arr) == 0 {
			return nil, nil
		}
		min := toFloat(arr[0])
		for _, v := range arr[1:] {
			if f := toFloat(v); f < min {
				min = f
			}
		}
		return min, nil
	}
	
	// Handle multiple arguments
	min := toFloat(args[0])
	for _, v := range args[1:] {
		if f := toFloat(v); f < min {
			min = f
		}
	}
	return min, nil
}

func funcMax(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("max requires at least 1 argument")
	}
	
	// Handle array input
	if arr, ok := args[0].([]interface{}); ok {
		if len(arr) == 0 {
			return nil, nil
		}
		max := toFloat(arr[0])
		for _, v := range arr[1:] {
			if f := toFloat(v); f > max {
				max = f
			}
		}
		return max, nil
	}
	
	// Handle multiple arguments
	max := toFloat(args[0])
	for _, v := range args[1:] {
		if f := toFloat(v); f > max {
			max = f
		}
	}
	return max, nil
}

func funcSum(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("sum requires at least 1 argument")
	}
	
	var sum float64
	
	if arr, ok := args[0].([]interface{}); ok {
		for _, v := range arr {
			sum += toFloat(v)
		}
	} else {
		for _, v := range args {
			sum += toFloat(v)
		}
	}
	
	return sum, nil
}

func funcAvg(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("avg requires at least 1 argument")
	}
	
	var sum float64
	var count int
	
	if arr, ok := args[0].([]interface{}); ok {
		for _, v := range arr {
			sum += toFloat(v)
			count++
		}
	} else {
		for _, v := range args {
			sum += toFloat(v)
			count++
		}
	}
	
	if count == 0 {
		return 0, nil
	}
	
	return sum / float64(count), nil
}

// Date functions

func funcNow(args ...interface{}) (interface{}, error) {
	return time.Now().Format(time.RFC3339), nil
}

func funcFormatDate(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("formatDate requires 2 arguments")
	}
	
	dateStr := fmt.Sprintf("%v", args[0])
	format := fmt.Sprintf("%v", args[1])
	
	// Parse the date
	t, err := parseAnyDate(dateStr)
	if err != nil {
		return nil, err
	}
	
	// Convert format from common patterns to Go format
	goFormat := convertDateFormat(format)
	return t.Format(goFormat), nil
}

func funcParseDate(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("parseDate requires 1 argument")
	}
	
	dateStr := fmt.Sprintf("%v", args[0])
	t, err := parseAnyDate(dateStr)
	if err != nil {
		return nil, err
	}
	
	return t.Format(time.RFC3339), nil
}

func funcAddDays(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("addDays requires 2 arguments")
	}
	
	dateStr := fmt.Sprintf("%v", args[0])
	days := toInt(args[1])
	
	t, err := parseAnyDate(dateStr)
	if err != nil {
		return nil, err
	}
	
	return t.AddDate(0, 0, days).Format(time.RFC3339), nil
}

func funcAddHours(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("addHours requires 2 arguments")
	}
	
	dateStr := fmt.Sprintf("%v", args[0])
	hours := toInt(args[1])
	
	t, err := parseAnyDate(dateStr)
	if err != nil {
		return nil, err
	}
	
	return t.Add(time.Duration(hours) * time.Hour).Format(time.RFC3339), nil
}

// JSON functions

func funcToJSON(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("toJson requires 1 argument")
	}
	b, err := json.Marshal(args[0])
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func funcFromJSON(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("fromJson requires 1 argument")
	}
	s := fmt.Sprintf("%v", args[0])
	var result interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func funcKeys(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("keys requires 1 argument")
	}
	m, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("argument must be an object")
	}
	keys := make([]interface{}, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys, nil
}

func funcValues(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("values requires 1 argument")
	}
	m, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("argument must be an object")
	}
	values := make([]interface{}, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values, nil
}

// Array functions

func funcFirst(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("first requires 1 argument")
	}
	arr, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("argument must be an array")
	}
	if len(arr) == 0 {
		return nil, nil
	}
	return arr[0], nil
}

func funcLast(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("last requires 1 argument")
	}
	arr, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("argument must be an array")
	}
	if len(arr) == 0 {
		return nil, nil
	}
	return arr[len(arr)-1], nil
}

func funcCount(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("count requires 1 argument")
	}
	arr, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("argument must be an array")
	}
	return len(arr), nil
}

func funcReverse(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("reverse requires 1 argument")
	}
	arr, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("argument must be an array")
	}
	result := make([]interface{}, len(arr))
	for i, v := range arr {
		result[len(arr)-1-i] = v
	}
	return result, nil
}

func funcSort(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("sort requires 1 argument")
	}
	arr, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("argument must be an array")
	}
	
	result := make([]interface{}, len(arr))
	copy(result, arr)
	
	sort.Slice(result, func(i, j int) bool {
		return fmt.Sprintf("%v", result[i]) < fmt.Sprintf("%v", result[j])
	})
	
	return result, nil
}

func funcUnique(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("unique requires 1 argument")
	}
	arr, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("argument must be an array")
	}
	
	seen := make(map[string]bool)
	result := make([]interface{}, 0)
	
	for _, v := range arr {
		key := fmt.Sprintf("%v", v)
		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}
	
	return result, nil
}

func funcFilter(args ...interface{}) (interface{}, error) {
	// Simplified filter - just removes nil/empty values
	if len(args) < 1 {
		return nil, fmt.Errorf("filter requires 1 argument")
	}
	arr, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("argument must be an array")
	}
	
	result := make([]interface{}, 0)
	for _, v := range arr {
		if v != nil && v != "" {
			result = append(result, v)
		}
	}
	
	return result, nil
}

func funcMap(args ...interface{}) (interface{}, error) {
	// Simplified map - extracts a field from array of objects
	if len(args) < 2 {
		return nil, fmt.Errorf("map requires 2 arguments")
	}
	arr, ok := args[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("first argument must be an array")
	}
	field := fmt.Sprintf("%v", args[1])
	
	result := make([]interface{}, len(arr))
	for i, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result[i] = m[field]
		}
	}
	
	return result, nil
}

// Type functions

func funcToString(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("toString requires 1 argument")
	}
	return fmt.Sprintf("%v", args[0]), nil
}

func funcToNumber(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("toNumber requires 1 argument")
	}
	return toFloat(args[0]), nil
}

func funcToBoolean(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("toBoolean requires 1 argument")
	}
	return toBool(args[0]), nil
}

func funcIsNull(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return true, nil
	}
	return args[0] == nil, nil
}

func funcIsEmpty(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return true, nil
	}
	v := args[0]
	if v == nil {
		return true, nil
	}
	switch val := v.(type) {
	case string:
		return val == "", nil
	case []interface{}:
		return len(val) == 0, nil
	case map[string]interface{}:
		return len(val) == 0, nil
	}
	return false, nil
}

func funcTypeof(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return "undefined", nil
	}
	v := args[0]
	if v == nil {
		return "null", nil
	}
	switch v.(type) {
	case string:
		return "string", nil
	case float64, int, int64:
		return "number", nil
	case bool:
		return "boolean", nil
	case []interface{}:
		return "array", nil
	case map[string]interface{}:
		return "object", nil
	default:
		return "unknown", nil
	}
}

// Utility functions

func funcIf(args ...interface{}) (interface{}, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("if requires 3 arguments")
	}
	condition := toBool(args[0])
	if condition {
		return args[1], nil
	}
	return args[2], nil
}

func funcDefault(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("default requires 2 arguments")
	}
	if args[0] == nil || args[0] == "" {
		return args[1], nil
	}
	return args[0], nil
}

func funcUUID(args ...interface{}) (interface{}, error) {
	return uuid.New().String(), nil
}

func funcBase64Encode(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("base64Encode requires 1 argument")
	}
	s := fmt.Sprintf("%v", args[0])
	return base64.StdEncoding.EncodeToString([]byte(s)), nil
}

func funcBase64Decode(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("base64Decode requires 1 argument")
	}
	s := fmt.Sprintf("%v", args[0])
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return string(decoded), nil
}

func funcHash(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("hash requires at least 1 argument")
	}
	s := fmt.Sprintf("%v", args[0])
	algo := "sha256"
	if len(args) >= 2 {
		algo = fmt.Sprintf("%v", args[1])
	}
	
	switch algo {
	case "md5":
		hash := md5.Sum([]byte(s))
		return hex.EncodeToString(hash[:]), nil
	case "sha256":
		hash := sha256.Sum256([]byte(s))
		return hex.EncodeToString(hash[:]), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algo)
	}
}

// Helper functions

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

func toInt(v interface{}) int {
	return int(toFloat(v))
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && val != "false" && val != "0"
	case float64:
		return val != 0
	case int:
		return val != 0
	case nil:
		return false
	default:
		return true
	}
}

func parseAnyDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
		"Jan 2, 2006",
		"January 2, 2006",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	
	// Try Unix timestamp
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}
	
	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

func convertDateFormat(format string) string {
	replacements := map[string]string{
		"YYYY": "2006",
		"YY":   "06",
		"MM":   "01",
		"DD":   "02",
		"HH":   "15",
		"mm":   "04",
		"ss":   "05",
		"SSS":  "000",
	}
	
	result := format
	for from, to := range replacements {
		result = strings.ReplaceAll(result, from, to)
	}
	return result
}
