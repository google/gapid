package pb

import "github.com/google/gapid/core/log"

func NewMessage(m *log.Message) {
	// &timestamp.Timestamp{Seconds: t.Unix(), Nanos: int32(t.Nanosecond())}

	//	for n := l.values; n != nil; n = n.parent {
	//		for name, value := range n.v {
	//			p := pod.NewValue(value)
	//			if p == nil {
	//				p = pod.NewValue(fmt.Sprint(value))
	//			}
	//			m.Values = append(m.Values, &Value{Name: name, Value: p})
	//		}
	//	}
}
