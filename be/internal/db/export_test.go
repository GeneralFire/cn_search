package db

// NewExecutorForTest — executor поверх querier; только для внешних тестов (package db_test).
func NewExecutorForTest(q querier) Executor { return executor{q: q} }
