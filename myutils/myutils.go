package myutils

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/shopspring/decimal"
	"github.com/thinkeridea/go-extend/exmath"
)

type MySyncMap[K any, V any] struct {
	smap sync.Map
}

func NewMySyncMap[K any, V any]() MySyncMap[K, V] {
	return MySyncMap[K, V]{
		smap: sync.Map{},
	}
}
func (m *MySyncMap[K, V]) Load(k K) (V, bool) {
	v, ok := m.smap.Load(k)

	if ok {
		return v.(V), true
	}
	var resv V
	return resv, false
}
func (m *MySyncMap[K, V]) Store(k K, v V) {
	m.smap.Store(k, v)
}

func (m *MySyncMap[K, V]) Delete(k K) {
	m.smap.Delete(k)
}
func (m *MySyncMap[K, V]) Range(f func(k K, v V) bool) {
	m.smap.Range(func(k, v any) bool {
		return f(k.(K), v.(V))
	})
}

func (m *MySyncMap[K, V]) Length() int {
	length := 0
	m.Range(func(k K, v V) bool {
		length += 1
		return true
	})
	return length
}

func (m *MySyncMap[K, V]) MapValues(f func(k K, v V) V) *MySyncMap[K, V] {
	var res = NewMySyncMap[K, V]()
	m.Range(func(k K, v V) bool {
		res.Store(k, f(k, v))
		return true
	})
	return &res
}
func GetPointer[T any](v T) *T {
	return &v
}

func GetArr1ExistAndArr2NotExist(arr1, arr2 []string) []string {
	res := []string{}
	for _, a1 := range arr1 {
		isExist := false
		for _, a2 := range arr2 {
			if a1 == a2 {
				isExist = true
				break
			}
		}
		if !isExist {
			res = append(res, a1)
		}
	}
	return res
}

func Round(x float64, unit int) float64 {
	return exmath.Round(x, unit)
}

func BigAddFloat64(x, y float64) float64 {
	result, _ := decimal.NewFromFloat(x).Add(decimal.NewFromFloat(y)).Round(16).Float64()
	return result
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func init() {
	rand.Seed(time.Now().UnixNano())
}
func RandStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// 快速排序 by copilot
func QuickSort[T any](list []T, compare func(a T, b T) bool) ([]T, error) {
	if len(list) < 2 {
		return list, nil
	}
	left, right := 0, len(list)-1
	pivot := rand.Int() % len(list)
	list[pivot], list[right] = list[right], list[pivot]
	for i := range list {
		if compare(list[i], list[right]) {
			list[left], list[i] = list[i], list[left]
			left++
		}
	}
	list[left], list[right] = list[right], list[left]
	QuickSort(list[:left], compare)
	QuickSort(list[left+1:], compare)
	return list, nil
}
func GetFloat64FromString(s string) float64 {
	res, _ := strconv.ParseFloat(s, 64)
	return res
}

func GetIntFromString(s string) int {
	res, _ := strconv.Atoi(s)
	return res
}
func GetInt64FromString(s string) int64 {
	res, _ := strconv.ParseInt(s, 10, 64)
	return res
}

func NewXID() string {
	return xid.New().String()
}

type BaseTypes interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 |
		~uint16 | ~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64
}

// 对旧map执行f处理返回新map，key值保持不变，v值变为新值
func MapValues[K BaseTypes, T any, NT any](m map[K]*T, f func(k K, v *T) NT) map[K]NT {
	res := map[K]NT{}
	for k, v := range m {
		res[k] = f(k, v)
	}
	return res
}

func MapValuesSameType[K BaseTypes, T any](m map[K]*T, f func(k K, v *T) T) map[K]T {
	return MapValues[K, T, T](m, f)
}

func MapList[T any](l []T, f func(v T) T) []T {
	newL := []T{}
	for _, v := range l {
		newL = append(newL, f(v))
	}
	return newL
}

func MapListConvert[T any, T2 any](l []T, f func(v T) T2) []T2 {
	newL := []T2{}
	for _, v := range l {
		newL = append(newL, f(v))
	}
	return newL
}

func ReverseSlice[T any](l []T) []T {
	newL := []T{}
	for i := len(l) - 1; i >= 0; i-- {
		newL = append(newL, l[i])
	}
	return newL
}

func Find[T any](slice []*T, f func(t *T) bool) *T {
	for _, v := range slice {
		if f(v) {
			return v
		}
	}
	return nil
}

// 高低优先级锁
// PriorityMutex 结构体包含了一个互斥锁和两个条件变量
type PriorityMutex struct {
	mu             *sync.Mutex
	highPriority   *sync.Cond
	lowPriority    *sync.Cond
	highWaiters    int
	lowWaiters     int
	activePriority string
}

// NewPriorityMutex 创建一个新的PriorityMutex
func NewPriorityMutex() *PriorityMutex {
	mu := sync.Mutex{}
	return &PriorityMutex{
		mu:             &mu,
		highPriority:   sync.NewCond(&mu),
		lowPriority:    sync.NewCond(&mu),
		highWaiters:    0,
		lowWaiters:     0,
		activePriority: "none",
	}
}

// LockHighPriority 高优先级任务调用此方法来请求锁
func (pm *PriorityMutex) LockHighPriority() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.highWaiters++
	for pm.activePriority == "low" {
		pm.highPriority.Wait()
	}
	pm.activePriority = "high"
	pm.highWaiters--
}

// UnlockHighPriority 高优先级任务完成后调用此方法来释放锁
func (pm *PriorityMutex) UnlockHighPriority() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.highWaiters > 0 {
		pm.highPriority.Signal()
	} else if pm.lowWaiters > 0 {
		pm.activePriority = "low"
		pm.lowPriority.Broadcast()
	} else {
		pm.activePriority = "none"
	}
}

// LockLowPriority 低优先级任务调用此方法来请求锁
func (pm *PriorityMutex) LockLowPriority() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.lowWaiters++
	for pm.activePriority == "high" {
		pm.lowPriority.Wait()
	}
	pm.activePriority = "low"
	pm.lowWaiters--
}

// TryLockLowPriority 尝试获取低优先级的锁
func (pm *PriorityMutex) TryLockLowPriority() bool {
	if pm.mu.TryLock() {
		defer pm.mu.Unlock()
	} else {
		return false
	}

	// 如果获取了锁，但有高优先级任务在等待，则释放锁并返回false
	if pm.activePriority != "none" {
		return false
	}

	// 如果没有高优先级任务在等待，设置活动优先级为低，并返回true
	pm.activePriority = "low"
	return true
}

// UnlockLowPriority 解锁低优先级的锁
func (pm *PriorityMutex) UnlockLowPriority() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.lowWaiters > 0 {
		pm.lowPriority.Signal()
	} else if pm.highWaiters > 0 {
		pm.activePriority = "high"
		pm.highPriority.Broadcast()
	} else {
		pm.activePriority = "none"
	}
}

// 计算小数点后有效位数的函数
func CountDecimalPlaces(str string) int {
	// 去除尾部的零
	str = strings.TrimRight(str, "0")
	// 分割字符串以获取小数部分
	parts := strings.Split(str, ".")
	// 如果有小数部分，则返回其长度
	if len(parts) == 2 {
		return len(parts[1])
	}
	// 如果没有小数部分，则返回0
	return 0
}

func CountDecimalPlaces2(n float64) int {
	if n == 1.0 {
		return 0
	} else if n > 1.0 {
		// 处理大于1的情况
		integerPart := int(math.Floor(n))
		digits := len(strconv.Itoa(integerPart))
		return 1 - digits
	} else {
		// 处理小于1的情况
		s := fmt.Sprintf("%.15f", n)
		parts := strings.Split(s, ".")
		if len(parts) != 2 {
			return 0
		}
		fractional := strings.TrimRight(parts[1], "0")
		if len(fractional) == 0 {
			return 0
		}
		return len(fractional)
	}
}

// 对买卖价格进行去重
func RemoveDuplicate[T any](dataList []T, getKey func(data T) string) []T {
	m := map[string]T{}
	for _, d := range dataList {
		m[getKey(d)] = d
	}
	var result []T
	for _, d := range m {
		result = append(result, d)
	}
	return result
}

type DecimalSortAsc []decimal.Decimal

func (a DecimalSortAsc) Len() int           { return len(a) }
func (a DecimalSortAsc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DecimalSortAsc) Less(i, j int) bool { return a[i].LessThan(a[j]) }

type DecimalSortDesc []decimal.Decimal

func (a DecimalSortDesc) Len() int           { return len(a) }
func (a DecimalSortDesc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DecimalSortDesc) Less(i, j int) bool { return a[i].GreaterThan(a[j]) }

func StringInArray(targetStr string, targetArr []string) bool {
	for _, s := range targetArr {
		if s == targetStr {
			return true
		}
	}
	return false
}

// 预排序版本（需要调用方保证数组已排序）
func StringInSortedArray(targetStr string, sortedArr []string) bool {
	index := sort.SearchStrings(sortedArr, targetStr)
	return index < len(sortedArr) && sortedArr[index] == targetStr
}
func Min[T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64](a, b T) T {
	if a < b {
		return a
	}
	return b
}
