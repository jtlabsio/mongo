package querybuilder

type LogicalOperator int

const (
	And LogicalOperator = iota // $and
	Not                        // $not
	Nor                        // $nor
	Or                         // $or
)

func (lo LogicalOperator) String() string {
	switch lo {
	case And:
		return "$and"
	case Not:
		return "$not"
	case Nor:
		return "$nor"
	case Or:
		return "$or"
	default:
		return ""
	}
}
