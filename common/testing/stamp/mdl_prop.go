package stamp

import (
	"reflect"
)

var (
	_          prop = Prop[bool]{}
	markerType      = reflect.TypeFor[Marker]()
)

type (
	PropContext struct{}
	Prop[T any] struct {
		val      T
		owner    modelWrapper
		name     string
		examples []example
	}
	Marker       Prop[bool]
	propAccessor interface {
		getVal() bool
		getName() string
		getOwner() modelWrapper
	}
	prop interface {
		String() string
		setName(string)
	}
	propTypes interface {
		// when updating this, make sure to also update the `typeCheckOption` switch below
		bool | int | string | ID
	}
	example struct {
		outcome any
		evalCtx *PropContext
	}
	propOption[T propTypes] func(*Prop[T])
)

func WithEventually() propOption[bool] {
	return func(prop *Prop[bool]) {
		// TODO
		// - need to keep track over a few seconds
		// - fail the model if it doesn't happen
		// - and wait at the end of the scenario until timeout
	}
}

//func WithExample[T propTypes](outcome T, record Record, records ...Record) propOption[T] {
//	return func(prop *Prop[T]) {
//		prop.examples = append(prop.examples, example{
//			outcome: outcome,
//			propCtx: newPropContext(append([]Record{record}, records...)...),
//		})
//	}
//}

func newPropContext() *PropContext {
	return &PropContext{}
}

func (p Prop[T]) setName(name string) {
	p.name = name
}

func (p Prop[T]) String() string {
	return p.name
}

func (p Prop[bool]) getVal() bool {
	return p.val
}

func (p Prop[T]) getName() string {
	return p.name
}

func (p Prop[T]) getOwner() modelWrapper {
	return p.owner
}

func (m Marker) getVal() bool {
	return m.val
}

func (m Marker) getName() string {
	return m.name
}

func (m Marker) getOwner() modelWrapper {
	return m.owner
}

func (m Marker) copy() Marker {
	return Marker{
		name: m.name,
	}
}

//func (p Prop[T]) Validate() error {
//	var failed bool
//	errs := make([]error, len(p.examples))
//	output := make([]any, len(p.examples))
//	for i, ex := range p.examples {
//		res, err := p.eval(ex.evalCtx)
//		if err != nil {
//			failed = true
//			errs[i] = err
//			continue
//		}
//		if res != ex.outcome {
//			failed = true
//			continue
//		}
//		output[i] = res
//	}
//	if failed {
//		var sb strings.Builder
//		sb.WriteString(fmt.Sprintf("examples failed:\n"))
//		sb.WriteString("\n")
//		sb.WriteString(underlineStr("Type:\n"))
//		sb.WriteString(p.typeOf.String())
//		sb.WriteString("\n")
//		for i, ex := range p.examples {
//			sb.WriteString("\n")
//			sb.WriteString(underlineStr(fmt.Sprintf("Example #%d:\n", i+1)))
//			sb.WriteString(fmt.Sprintf("input: %v\n", ex.evalCtx))
//			if errs[i] != nil {
//				sb.WriteString(fmt.Sprintf("%s: %v\n", redStr("error"), errs[i]))
//				continue
//			}
//			sb.WriteString(fmt.Sprintf("output: %v\n", output[i]))
//			sb.WriteString(fmt.Sprintf("expected: %v\n", ex.outcome))
//		}
//		return errors.New(sb.String())
//	}
//	return nil
//}

//func (p Prop[T]) get() (T, *PropContext, error) {
//	evalCtx := p.owner.getPropCtx()
//	res, err := p.eval(evalCtx)
//	if res == nil {
//		var zero T
//		return zero, nil, err
//	}
//	return res.(T), evalCtx, err
//}

//func (p Prop[T]) Get() T {
//	res, _, err := p.get()
//	if err != nil {
//		panic(err)
//	}
//	return res
//}

//func (p Prop[T]) WaitGet(_ genContext) T {
//	timeout := time.After(2 * time.Second)           // TODO: take from genCtx
//	ticker := time.NewTicker(100 * time.Millisecond) // TODO: take from genCtx
//	defer ticker.Stop()
//
//	var lastErr error
//	for {
//		select {
//		case <-ticker.C:
//			res, _, err := p.get()
//			if err == nil {
//				return res
//			}
//			lastErr = err
//		case <-timeout:
//			panic(fmt.Errorf("prop %q failed to eval after timeout: %w", p.String(), lastErr))
//		}
//	}
//}

func (m *Marker) Mark() {
	m.val = true
}
