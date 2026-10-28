package model

import "testing"

func TestTradeArithmeticAndSalePolicy(t *testing.T) {
	for value, want := range map[int]int{15: 3, 20: 4, 120: 24, 1000: 200} {
		if ItemSellPrice(value) != want {
			t.Fatalf("value=%d", value)
		}
	}
	for _, n := range []int{-1, 0, 10001, int(^uint(0) >> 1)} {
		if ValidTransactionQuantity(n) {
			t.Fatalf("invalid quantity %d", n)
		}
	}
	if _, err := CheckedProduct(int(^uint(0)>>1), 2); err == nil {
		t.Fatal("overflow product accepted")
	}
	if _, err := CheckedAdd(int(^uint(0)>>1), 1); err == nil {
		t.Fatal("overflow sum accepted")
	}
}
