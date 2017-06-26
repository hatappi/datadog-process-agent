package aggregator

import (
	// stdlib
	"math/rand"
	"testing"
	"time"

	// 3p
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultHistogramSampling(t *testing.T) {
	// Initialize default histogram
	mHistogram := Histogram{}

	// Empty flush
	_, err := mHistogram.flush(50)
	assert.NotNil(t, err)

	// Add samples
	mHistogram.addSample(&MetricSample{Value: 1}, 50)
	mHistogram.addSample(&MetricSample{Value: 10}, 51)
	mHistogram.addSample(&MetricSample{Value: 4}, 55)
	mHistogram.addSample(&MetricSample{Value: 5}, 55)
	mHistogram.addSample(&MetricSample{Value: 2}, 55)
	mHistogram.addSample(&MetricSample{Value: 2}, 55)

	series, err := mHistogram.flush(60)
	assert.Nil(t, err)
	if assert.Len(t, series, 5) {
		for _, serie := range series {
			assert.Len(t, serie.Points, 1)
			assert.EqualValues(t, 60, serie.Points[0].Ts)
		}
		assert.InEpsilon(t, 10, series[0].Points[0].Value, epsilon)     // max
		assert.Equal(t, ".max", series[0].nameSuffix)                   // max
		assert.InEpsilon(t, 2, series[1].Points[0].Value, epsilon)      // median
		assert.Equal(t, ".median", series[1].nameSuffix)                // median
		assert.InEpsilon(t, 12./3., series[2].Points[0].Value, epsilon) // avg
		assert.Equal(t, ".avg", series[2].nameSuffix)                   // avg
		assert.InEpsilon(t, 6, series[3].Points[0].Value, epsilon)      // count
		assert.Equal(t, ".count", series[3].nameSuffix)                 // count
		assert.InEpsilon(t, 10, series[4].Points[0].Value, epsilon)     // 0.95
		assert.Equal(t, ".95percentile", series[4].nameSuffix)          // 0.95
	}

	_, err = mHistogram.flush(61)
	assert.NotNil(t, err)
}

func TestCustomHistogramSampling(t *testing.T) {
	// Initialize custom histogram, with an invalid aggregate
	mHistogram := Histogram{}
	mHistogram.configure([]string{"min", "sum", "invalid"}, []int{})

	// Empty flush
	_, err := mHistogram.flush(50)
	assert.NotNil(t, err)

	// Add samples
	mHistogram.addSample(&MetricSample{Value: 1}, 50)
	mHistogram.addSample(&MetricSample{Value: 10}, 51)
	mHistogram.addSample(&MetricSample{Value: 4}, 55)
	mHistogram.addSample(&MetricSample{Value: 5}, 55)
	mHistogram.addSample(&MetricSample{Value: 2}, 55)
	mHistogram.addSample(&MetricSample{Value: 2}, 55)

	series, err := mHistogram.flush(60)
	assert.Nil(t, err)
	if assert.Len(t, series, 2) {
		// Only 2 series are returned (the invalid aggregate is ignored)
		for _, serie := range series {
			assert.Len(t, serie.Points, 1)
			assert.EqualValues(t, 60, serie.Points[0].Ts)
		}
		assert.InEpsilon(t, 1, series[0].Points[0].Value, epsilon)            // min
		assert.Equal(t, ".min", series[0].nameSuffix)                         // min
		assert.InEpsilon(t, 1+10+4+5+2+2, series[1].Points[0].Value, epsilon) // sum
		assert.Equal(t, ".sum", series[1].nameSuffix)                         // sum
	}

	_, err = mHistogram.flush(61)
	assert.NotNil(t, err)
}

func shuffle(slice []float64) {
	t := time.Now()
	rand.Seed(int64(t.Nanosecond()))

	for i := len(slice) - 1; i > 0; i-- {
		j := rand.Intn(i)
		slice[i], slice[j] = slice[j], slice[i]
	}
}

func TestHistogramPercentiles(t *testing.T) {
	// Initialize custom histogram
	mHistogram := Histogram{}
	mHistogram.configure([]string{"max", "median", "avg", "count", "min"}, []int{95, 80})

	// Empty flush
	_, err := mHistogram.flush(50)
	assert.NotNil(t, err)

	// Sample 20 times all numbers between 1 and 100.
	// This means our percentiles should be relatively close to themselves.
	var percentiles []float64
	for i := 1; i <= 100; i++ {
		percentiles = append(percentiles, float64(i))
	}
	shuffle(percentiles) // in place
	for _, p := range percentiles {
		for j := 0; j < 20; j++ {
			mHistogram.addSample(&MetricSample{Value: p}, 50)
		}
	}

	series, err := mHistogram.flush(60)
	assert.Nil(t, err)
	if assert.Len(t, series, 7) {
		for _, serie := range series {
			assert.Len(t, serie.Points, 1)
			assert.EqualValues(t, 60, serie.Points[0].Ts)
		}
		assert.InEpsilon(t, 100, series[0].Points[0].Value, epsilon)    // max
		assert.Equal(t, ".max", series[0].nameSuffix)                   // max
		assert.InEpsilon(t, 50, series[1].Points[0].Value, epsilon)     // median
		assert.Equal(t, ".median", series[1].nameSuffix)                // median
		assert.InEpsilon(t, 50, series[2].Points[0].Value, epsilon)     // avg
		assert.Equal(t, ".avg", series[2].nameSuffix)                   // avg
		assert.InEpsilon(t, 100*20, series[3].Points[0].Value, epsilon) // count
		assert.Equal(t, ".count", series[3].nameSuffix)                 // count
		assert.InEpsilon(t, 1, series[4].Points[0].Value, epsilon)      // min
		assert.Equal(t, ".min", series[4].nameSuffix)                   // min
		assert.InEpsilon(t, 80, series[5].Points[0].Value, epsilon)     // 0.80
		assert.Equal(t, ".80percentile", series[5].nameSuffix)          // 0.80
		assert.InEpsilon(t, 95, series[6].Points[0].Value, epsilon)     // 0.95
		assert.Equal(t, ".95percentile", series[6].nameSuffix)          // 0.95
	}

	_, err = mHistogram.flush(61)
	assert.NotNil(t, err)
}

func TestHistogramSampleRate(t *testing.T) {
	mHistogram := Histogram{}
	mHistogram.configure([]string{"max", "min", "median", "avg", "sum", "count"}, []int{20, 95, 80})

	mHistogram.addSample(&MetricSample{Value: 1}, 50)
	mHistogram.addSample(&MetricSample{Value: 2, SampleRate: 0.5}, 50)
	mHistogram.addSample(&MetricSample{Value: 3, SampleRate: 0.2}, 50)
	mHistogram.addSample(&MetricSample{Value: 10, SampleRate: 0.5}, 50)

	series, err := mHistogram.flush(60)
	assert.Nil(t, err)
	require.Len(t, series, 9)

	for _, serie := range series {
		assert.Len(t, serie.Points, 1)
		assert.EqualValues(t, 60, serie.Points[0].Ts)
	}
	assert.InEpsilon(t, 10, series[0].Points[0].Value, epsilon) // max
	assert.Equal(t, ".max", series[0].nameSuffix)               // max
	assert.InEpsilon(t, 1, series[1].Points[0].Value, epsilon)  // min
	assert.Equal(t, ".min", series[1].nameSuffix)               // min
	assert.InEpsilon(t, 3, series[2].Points[0].Value, epsilon)  // median
	assert.Equal(t, ".median", series[2].nameSuffix)            // median
	assert.InEpsilon(t, 4, series[3].Points[0].Value, epsilon)  // avg
	assert.Equal(t, ".avg", series[3].nameSuffix)               // avg
	assert.InEpsilon(t, 40, series[4].Points[0].Value, epsilon) // sum
	assert.Equal(t, ".sum", series[4].nameSuffix)               // sum
	assert.InEpsilon(t, 10, series[5].Points[0].Value, epsilon) // count
	assert.Equal(t, ".count", series[5].nameSuffix)             // count
	assert.InEpsilon(t, 2, series[6].Points[0].Value, epsilon)  // 0.20
	assert.Equal(t, ".20percentile", series[6].nameSuffix)      // 0.20
	assert.InEpsilon(t, 3, series[7].Points[0].Value, epsilon)  // 0.80
	assert.Equal(t, ".80percentile", series[7].nameSuffix)      // 0.80
	assert.InEpsilon(t, 10, series[8].Points[0].Value, epsilon) // 0.95
	assert.Equal(t, ".95percentile", series[8].nameSuffix)      // 0.95

	_, err = mHistogram.flush(61)
	assert.NotNil(t, err)
}

func TestHistogramReset(t *testing.T) {
	mHistogram := Histogram{}
	mHistogram.configure([]string{"max", "min", "median", "avg", "sum", "count"}, []int{20, 95, 80})

	mHistogram.addSample(&MetricSample{Value: 1}, 50)
	mHistogram.addSample(&MetricSample{Value: 2, SampleRate: 0.5}, 50)
	series, err := mHistogram.flush(60)
	assert.Nil(t, err)

	mHistogram.addSample(&MetricSample{Value: 10}, 50)
	series, err = mHistogram.flush(70)
	assert.Nil(t, err)
	require.Len(t, series, 9)

	for _, serie := range series {
		assert.Len(t, serie.Points, 1)
		assert.EqualValues(t, 70, serie.Points[0].Ts)
	}
	assert.InEpsilon(t, 10, series[0].Points[0].Value, epsilon) // max
	assert.Equal(t, ".max", series[0].nameSuffix)               // max
	assert.InEpsilon(t, 10, series[1].Points[0].Value, epsilon) // min
	assert.Equal(t, ".min", series[1].nameSuffix)               // min
	assert.InEpsilon(t, 10, series[2].Points[0].Value, epsilon) // median
	assert.Equal(t, ".median", series[2].nameSuffix)            // median
	assert.InEpsilon(t, 10, series[3].Points[0].Value, epsilon) // avg
	assert.Equal(t, ".avg", series[3].nameSuffix)               // avg
	assert.InEpsilon(t, 10, series[4].Points[0].Value, epsilon) // sum
	assert.Equal(t, ".sum", series[4].nameSuffix)               // sum
	assert.InEpsilon(t, 1, series[5].Points[0].Value, epsilon)  // count
	assert.Equal(t, ".count", series[5].nameSuffix)             // count
	assert.InEpsilon(t, 10, series[6].Points[0].Value, epsilon) // 0.20
	assert.Equal(t, ".20percentile", series[6].nameSuffix)      // 0.20
	assert.InEpsilon(t, 10, series[7].Points[0].Value, epsilon) // 0.80
	assert.Equal(t, ".80percentile", series[7].nameSuffix)      // 0.80
	assert.InEpsilon(t, 10, series[8].Points[0].Value, epsilon) // 0.95
	assert.Equal(t, ".95percentile", series[8].nameSuffix)      // 0.95

	_, err = mHistogram.flush(71)
	assert.NotNil(t, err)
}

//
// Benchmark
//

func benchHistogram(b *testing.B, number int, sampleRate float64) {
	for n := 0; n < b.N; n++ {
		h := Histogram{}
		h.configure([]string{"max", "min", "median", "avg", "sum", "count"}, []int{20, 95, 80})
		m := MetricSample{Value: 21, SampleRate: sampleRate}

		for i := 0; i < number; i++ {
			h.addSample(&m, 10)
		}
		h.flush(10)
	}
}

func BenchmarkHistogram2SampleRate1(b *testing.B) {
	benchHistogram(b, 2, 1.0)
}

func BenchmarkHistogram10SampleRate1(b *testing.B) {
	benchHistogram(b, 10, 1.0)
}

func BenchmarkHistogram100SampleRate1(b *testing.B) {
	benchHistogram(b, 100, 1.0)
}

func BenchmarkHistogram1000SampleRate1(b *testing.B) {
	benchHistogram(b, 1000, 1.0)
}

func BenchmarkHistogram10000SampleRate1(b *testing.B) {
	benchHistogram(b, 10000, 1.0)
}

func BenchmarkHistogram100000SampleRate1(b *testing.B) {
	benchHistogram(b, 100000, 1.0)
}

func BenchmarkHistogram2SampleRate05(b *testing.B) {
	benchHistogram(b, 2, 0.5)
}

func BenchmarkHistogram10SampleRate05(b *testing.B) {
	benchHistogram(b, 10, 0.5)
}

func BenchmarkHistogram100SampleRate05(b *testing.B) {
	benchHistogram(b, 100, 0.5)
}

func BenchmarkHistogram1000SampleRate05(b *testing.B) {
	benchHistogram(b, 1000, 0.5)
}

func BenchmarkHistogram10000SampleRate05(b *testing.B) {
	benchHistogram(b, 10000, 0.5)
}

func BenchmarkHistogram100000SampleRate05(b *testing.B) {
	benchHistogram(b, 100000, 0.5)
}

func BenchmarkHistogram2SampleRate02(b *testing.B) {
	benchHistogram(b, 2, 0.2)
}

func BenchmarkHistogram10SampleRate02(b *testing.B) {
	benchHistogram(b, 10, 0.2)
}

func BenchmarkHistogram100SampleRate02(b *testing.B) {
	benchHistogram(b, 100, 0.2)
}

func BenchmarkHistogram1000SampleRate02(b *testing.B) {
	benchHistogram(b, 1000, 0.2)
}

func BenchmarkHistogram10000SampleRate02(b *testing.B) {
	benchHistogram(b, 10000, 0.2)
}

func BenchmarkHistogram100000SampleRate02(b *testing.B) {
	benchHistogram(b, 100000, 0.2)
}
