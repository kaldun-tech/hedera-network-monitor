package alerting

import (
	"testing"
)

// TestEvaluateCondition_GreaterThan tests the > condition
func TestEvaluateCondition_GreaterThan(t *testing.T) {
	rule := &AlertRule{
		ID:        "gt_rule",
		Name:      "Greater Than Test",
		Condition: ">",
		Threshold: 100.0,
	}

	// Test: 150 > 100 (true)
	if !rule.EvaluateCondition(150.0, 0, false) {
		t.Error("expected > condition to be true for 150 > 100")
	}

	// Test: 100 > 100 (false)
	if rule.EvaluateCondition(100.0, 0, false) {
		t.Error("expected > condition to be false for 100 > 100")
	}

	// Test: 50 > 100 (false)
	if rule.EvaluateCondition(50.0, 0, false) {
		t.Error("expected > condition to be false for 50 > 100")
	}
}

// TestEvaluateCondition_LessThan tests the < condition
func TestEvaluateCondition_LessThan(t *testing.T) {
	rule := &AlertRule{
		ID:        "lt_rule",
		Name:      "Less Than Test",
		Condition: "<",
		Threshold: 100.0,
	}

	// Test: 50 < 100 (true)
	if !rule.EvaluateCondition(50.0, 0, false) {
		t.Error("expected < condition to be true for 50 < 100")
	}

	// Test: 100 < 100 (false)
	if rule.EvaluateCondition(100.0, 0, false) {
		t.Error("expected < condition to be false for 100 < 100")
	}

	// Test: 150 < 100 (false)
	if rule.EvaluateCondition(150.0, 0, false) {
		t.Error("expected < condition to be false for 150 < 100")
	}
}

// TestEvaluateCondition_GreaterThanOrEqual tests the >= condition
func TestEvaluateCondition_GreaterThanOrEqual(t *testing.T) {
	rule := &AlertRule{
		ID:        "gte_rule",
		Name:      "Greater Than Or Equal Test",
		Condition: ">=",
		Threshold: 100.0,
	}

	// Test: 150 >= 100 (true)
	if !rule.EvaluateCondition(150.0, 0, false) {
		t.Error("expected >= condition to be true for 150 >= 100")
	}

	// Test: 100 >= 100 (true) - This is the key difference from >
	if !rule.EvaluateCondition(100.0, 0, false) {
		t.Error("expected >= condition to be true for 100 >= 100")
	}

	// Test: 50 >= 100 (false)
	if rule.EvaluateCondition(50.0, 0, false) {
		t.Error("expected >= condition to be false for 50 >= 100")
	}
}

// TestEvaluateCondition_LessThanOrEqual tests the <= condition
func TestEvaluateCondition_LessThanOrEqual(t *testing.T) {
	rule := &AlertRule{
		ID:        "lte_rule",
		Name:      "Less Than Or Equal Test",
		Condition: "<=",
		Threshold: 100.0,
	}

	// Test: 50 <= 100 (true)
	if !rule.EvaluateCondition(50.0, 0, false) {
		t.Error("expected <= condition to be true for 50 <= 100")
	}

	// Test: 100 <= 100 (true) - This is the key difference from <
	if !rule.EvaluateCondition(100.0, 0, false) {
		t.Error("expected <= condition to be true for 100 <= 100")
	}

	// Test: 150 <= 100 (false)
	if rule.EvaluateCondition(150.0, 0, false) {
		t.Error("expected <= condition to be false for 150 <= 100")
	}
}

// TestEvaluateCondition_Equal tests the == condition
func TestEvaluateCondition_Equal(t *testing.T) {
	rule := &AlertRule{
		ID:        "eq_rule",
		Name:      "Equal Test",
		Condition: "==",
		Threshold: 100.0,
	}

	// Test: 100 == 100 (true)
	if !rule.EvaluateCondition(100.0, 0, false) {
		t.Error("expected == condition to be true for 100 == 100")
	}

	// Test: 150 == 100 (false)
	if rule.EvaluateCondition(150.0, 0, false) {
		t.Error("expected == condition to be false for 150 == 100")
	}
}

// TestEvaluateCondition_NotEqual tests the != condition
func TestEvaluateCondition_NotEqual(t *testing.T) {
	rule := &AlertRule{
		ID:        "ne_rule",
		Name:      "Not Equal Test",
		Condition: "!=",
		Threshold: 100.0,
	}

	// Test: 150 != 100 (true)
	if !rule.EvaluateCondition(150.0, 0, false) {
		t.Error("expected != condition to be true for 150 != 100")
	}

	// Test: 100 != 100 (false)
	if rule.EvaluateCondition(100.0, 0, false) {
		t.Error("expected != condition to be false for 100 != 100")
	}
}

// TestEvaluateCondition_Changed tests the changed condition (value differs from previous)
func TestEvaluateCondition_Changed(t *testing.T) {
	rule := &AlertRule{
		ID:        "changed_rule",
		Name:      "Changed Test",
		Condition: "changed",
	}

	// Test 1: Value changed (100 -> 150) with previous value
	if !rule.EvaluateCondition(150.0, 100.0, true) {
		t.Error("expected changed condition to be true when value differs")
	}

	// Test 2: Value unchanged (100 -> 100)
	if rule.EvaluateCondition(100.0, 100.0, true) {
		t.Error("expected changed condition to be false when value is same")
	}

	// Test 3: Value changed back (150 -> 100)
	if !rule.EvaluateCondition(100.0, 150.0, true) {
		t.Error("expected changed condition to be true when value changes back")
	}

	// Test 4: First metric (no previous value) should NOT trigger
	if rule.EvaluateCondition(100.0, 0.0, false) {
		t.Error("expected changed condition to be false for first metric (no previous value)")
	}
}

// TestEvaluateCondition_Increased tests the increased condition (value > previous)
func TestEvaluateCondition_Increased(t *testing.T) {
	rule := &AlertRule{
		ID:        "increased_rule",
		Name:      "Increased Test",
		Condition: "increased",
	}

	// Test 1: Value increased (100 -> 150)
	if !rule.EvaluateCondition(150.0, 100.0, true) {
		t.Error("expected increased condition to be true when value goes up")
	}

	// Test 2: Value decreased (150 -> 100)
	if rule.EvaluateCondition(100.0, 150.0, true) {
		t.Error("expected increased condition to be false when value goes down")
	}

	// Test 3: Value unchanged (100 -> 100)
	if rule.EvaluateCondition(100.0, 100.0, true) {
		t.Error("expected increased condition to be false when value is same")
	}

	// Test 4: Increase from zero (0 -> 50)
	if !rule.EvaluateCondition(50.0, 0.0, true) {
		t.Error("expected increased condition to be true when increasing from zero")
	}

	// Test 5: First metric (no previous value) should NOT trigger
	if rule.EvaluateCondition(100.0, 0.0, false) {
		t.Error("expected increased condition to be false for first metric (no previous value)")
	}
}

// TestEvaluateCondition_Decreased tests the decreased condition (value < previous)
func TestEvaluateCondition_Decreased(t *testing.T) {
	rule := &AlertRule{
		ID:        "decreased_rule",
		Name:      "Decreased Test",
		Condition: "decreased",
	}

	// Test 1: Value decreased (150 -> 100)
	if !rule.EvaluateCondition(100.0, 150.0, true) {
		t.Error("expected decreased condition to be true when value goes down")
	}

	// Test 2: Value increased (100 -> 150)
	if rule.EvaluateCondition(150.0, 100.0, true) {
		t.Error("expected decreased condition to be false when value goes up")
	}

	// Test 3: Value unchanged (100 -> 100)
	if rule.EvaluateCondition(100.0, 100.0, true) {
		t.Error("expected decreased condition to be false when value is same")
	}

	// Test 4: Decrease to zero (50 -> 0)
	if !rule.EvaluateCondition(0.0, 50.0, true) {
		t.Error("expected decreased condition to be true when decreasing to zero")
	}

	// Test 5: First metric (no previous value) should NOT trigger
	if rule.EvaluateCondition(100.0, 0.0, false) {
		t.Error("expected decreased condition to be false for first metric (no previous value)")
	}
}
