package model

import "errors"

const MaxTransactionAmount = 10000

func ValidTransactionQuantity(n int) bool { return n >= 1 && n <= MaxTransactionAmount }
func ItemSellPrice(value int) int         { return max(0, value/5) }

func CheckedAdd(a, b int) (int, error) {
	if a < 0 || b < 0 || a > int(^uint(0)>>1)-b {
		return 0, errors.New("数值超出允许范围")
	}
	return a + b, nil
}
func CheckedProduct(a, b int) (int, error) {
	if a < 0 || b < 0 || (b > 0 && a > int(^uint(0)>>1)/b) {
		return 0, errors.New("数值超出允许范围")
	}
	return a * b, nil
}
