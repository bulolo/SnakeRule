package evaluator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/szplanner/snakerule/internal/domain"
)

// Evaluate applies a deterministic condition tree to a fact map.
func Evaluate(condition domain.Condition, facts map[string]any) (bool, error) {
	if err := Validate(condition); err != nil {
		return false, err
	}
	return evaluate(condition, facts)
}

func evaluate(condition domain.Condition, facts map[string]any) (bool, error) {
	branches := 0
	if condition.All != nil {
		branches++
	}
	if condition.Any != nil {
		branches++
	}
	if condition.Not != nil {
		branches++
	}
	if condition.Fact != "" {
		branches++
	}
	if branches != 1 {
		return false, fmt.Errorf("condition must contain exactly one of all, any, not, or fact")
	}
	if condition.All != nil {
		for _, child := range condition.All {
			ok, err := evaluate(child, facts)
			if err != nil || !ok {
				return ok, err
			}
		}
		return true, nil
	}
	if condition.Any != nil {
		for _, child := range condition.Any {
			ok, err := evaluate(child, facts)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	if condition.Not != nil {
		ok, err := evaluate(*condition.Not, facts)
		return !ok, err
	}
	actual, exists := lookup(facts, condition.Fact)
	return compare(actual, exists, condition.Operator, condition.Value)
}

func Validate(condition domain.Condition) error {
	branches := 0
	if condition.All != nil {
		branches++
	}
	if condition.Any != nil {
		branches++
	}
	if condition.Not != nil {
		branches++
	}
	if condition.Fact != "" {
		branches++
	}
	if branches != 1 {
		return fmt.Errorf("condition must contain exactly one of all, any, not, or fact")
	}
	if condition.All != nil {
		for _, child := range condition.All {
			if err := Validate(child); err != nil {
				return err
			}
		}
		return nil
	}
	if condition.Any != nil {
		for _, child := range condition.Any {
			if err := Validate(child); err != nil {
				return err
			}
		}
		return nil
	}
	if condition.Not != nil {
		return Validate(*condition.Not)
	}
	if strings.TrimSpace(condition.Fact) == "" {
		return fmt.Errorf("fact path must not be empty")
	}
	switch condition.Operator {
	case "equal", "eq", "notEqual", "ne", "greaterThan", "gt", "greaterThanInclusive", "gte", "lessThan", "lt", "lessThanInclusive", "lte", "in", "notIn", "contains", "isNull", "isNotNull":
		return nil
	default:
		return fmt.Errorf("unsupported operator %q", condition.Operator)
	}
}

func lookup(facts map[string]any, path string) (any, bool) {
	var value any = facts
	for _, part := range strings.Split(path, ".") {
		m, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return value, true
}

func compare(actual any, exists bool, operator string, expected any) (bool, error) {
	switch operator {
	case "isNull":
		return !exists || isNil(actual), nil
	case "isNotNull":
		return exists && !isNil(actual), nil
	case "equal", "eq":
		return exists && equal(actual, expected), nil
	case "notEqual", "ne":
		return exists && !equal(actual, expected), nil
	case "in", "notIn":
		values, ok := expected.([]any)
		if !ok {
			return false, fmt.Errorf("operator %s expects an array value", operator)
		}
		found := false
		for _, candidate := range values {
			if equal(actual, candidate) {
				found = true
				break
			}
		}
		if operator == "notIn" {
			return exists && !found, nil
		}
		return exists && found, nil
	case "contains":
		if !exists {
			return false, nil
		}
		switch value := actual.(type) {
		case string:
			needle, ok := expected.(string)
			if !ok {
				return false, fmt.Errorf("contains expects a string value")
			}
			return strings.Contains(value, needle), nil
		case []any:
			for _, item := range value {
				if equal(item, expected) {
					return true, nil
				}
			}
			return false, nil
		default:
			return false, fmt.Errorf("contains is not supported for %T", actual)
		}
	case "greaterThan", "gt", "greaterThanInclusive", "gte", "lessThan", "lt", "lessThanInclusive", "lte":
		if !exists {
			return false, nil
		}
		a, err := number(actual)
		if err != nil {
			return false, fmt.Errorf("fact value: %w", err)
		}
		b, err := number(expected)
		if err != nil {
			return false, fmt.Errorf("comparison value: %w", err)
		}
		switch operator {
		case "greaterThan", "gt":
			return a > b, nil
		case "greaterThanInclusive", "gte":
			return a >= b, nil
		case "lessThan", "lt":
			return a < b, nil
		default:
			return a <= b, nil
		}
	default:
		return false, fmt.Errorf("unsupported operator %q", operator)
	}
}

func number(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case json.Number:
		return strconv.ParseFloat(string(v), 64)
	default:
		return 0, fmt.Errorf("expected a number, got %T", value)
	}
}

func isNil(value any) bool { return value == nil }

func equal(a, b any) bool {
	if left, err := number(a); err == nil {
		if right, err := number(b); err == nil {
			return left == right
		}
	}
	return reflect.DeepEqual(a, b)
}
