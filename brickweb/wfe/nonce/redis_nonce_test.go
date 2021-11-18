package nonce_test

/*import (
	"brick/brickweb/wfe/nonce"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func getRedisNoncer() nonce.NonceService {
	noncer := func() (non nonce.NonceService) {
		defer func() {
			x := recover()
			if x != nil {
				non = nonce.NewRedisNoncer("redis:6379")
			}
		}()
		non = nonce.NewRedisNoncer("localhost:6379")
		return
	}()
	return noncer
}

func TestCreateABunchOfNoncesRedis(t *testing.T) {
	nonceSvc := getRedisNoncer()
	size := 300
	nonces := make([]nonce.Nonce, size)
	for i := 0; i < size; i++ {
		nonces[i] = nonceSvc.Next()
	}
	//All nonces must be valid
	for i := 0; i < size; i++ {
		if !nonceSvc.Valid(nonces[i]) {
			t.Fail()
		}
	}
	//Nonce of those nonces can still be valid
	for i := 0; i < size; i++ {
		if nonceSvc.Valid(nonces[i]) {
			t.Fail()
		}
	}
}

func TestCreateParallelNoncesRedis(t *testing.T) {
	noncer := getRedisNoncer()
	nonces := make([]nonce.Nonce, 5000)
	wg := sync.WaitGroup{}
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func(count int) {
			for j := 0; j < 1000; j++ {
				nonces[count*1000+j] = noncer.Next()
				ok := noncer.Valid(nonces[count*1000+j])
				if !ok {
					t.Errorf("Nonce #%d not valid: %s", count*1000+j, nonces[count*1000+j])
					t.Fail()
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	nonceMap := make(map[nonce.Nonce]time.Time)
	i := 0
	for _, n := range nonces {
		i++
		_, ok := nonceMap[n]
		if ok {
			t.Errorf("%s Nonce twice #%d", string(n), i)
		}
		nonceMap[n] = time.Now()
	}
}

func TestNonceTwiceRedis(t *testing.T) {
	noncer := getRedisNoncer()
	Convey("When a Nonce is created", t, func() {
		nonce := noncer.Next()
		Convey("It should be valid once", func() {
			So(noncer.Valid(nonce), ShouldBeTrue)
			Convey("But not twice", func() {
				So(noncer.Valid(nonce), ShouldBeFalse)
			})
		})
	})
}
*/
