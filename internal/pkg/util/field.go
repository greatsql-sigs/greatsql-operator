package util

import (
	"fmt"
	"reflect"
)

// Extractor 提供类型安全的反射字段访问方法
type Extractor struct {
	obj reflect.Value
}

// NewExtractor 创建一个 Extractor
func NewExtractor(obj any) (*Extractor, error) {
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil, fmt.Errorf("expected non-nil pointer")
	}
	return &Extractor{obj: v.Elem()}, nil
}

// Get 获取字段原始 interface 值
func (e *Extractor) Get(field string) (any, error) {
	f := e.obj.FieldByName(field)
	if !f.IsValid() {
		return nil, fmt.Errorf("field %q not found", field)
	}
	return f.Interface(), nil
}

// MustGet 获取字段原始 interface，失败 panic（谨慎使用）
func (e *Extractor) MustGet(field string) any {
	v, err := e.Get(field)
	if err != nil {
		panic(err)
	}
	return v
}

// GetString 获取 string 类型字段
func (e *Extractor) GetString(field string) (string, error) {
	val, err := e.Get(field)
	if err != nil {
		return "", err
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("field %q is not string", field)
	}
	return str, nil
}

// GetInt64Ptr 获取 int64 指针字段
func (e *Extractor) GetInt64Ptr(field string) (*int64, error) {
	val, err := e.Get(field)
	if err != nil {
		return nil, err
	}
	num, ok := val.(int64)
	if !ok {
		return nil, fmt.Errorf("field %q is not int64", field)
	}
	return &num, nil
}

// GetTyped 获取字段并断言为指定类型（作为独立函数实现）
func GetTyped[T any](e *Extractor, field string) (T, error) {
	var zero T
	val, err := e.Get(field)
	if err != nil {
		return zero, err
	}
	t, ok := val.(T)
	if !ok {
		return zero, fmt.Errorf("field %q is not of type %T", field, zero)
	}
	return t, nil
}
