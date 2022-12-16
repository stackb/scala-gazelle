// copied from https://github.com/Tufin/asciitree; Apache 2
package collections

type boxType int

const (
	Regular boxType = iota
	Last
	AfterLast
	Between
)

func (boxType boxType) String() string {
	switch boxType {
	case Regular:
		return "\u251c" // ├
	case Last:
		return "\u2514" // └
	case AfterLast:
		return " "
	case Between:
		return "\u2502" // │
	default:
		panic("invalid box type")
	}
}

func getBoxType(index int, len int) boxType {
	if index+1 == len {
		return Last
	} else if index+1 > len {
		return AfterLast
	}
	return Regular
}

func getBoxTypeExternal(index int, len int) boxType {
	if index+1 == len {
		return AfterLast
	}
	return Between
}

func getBoxPadding(root bool, boxType boxType) string {
	if root {
		return ""
	}

	return boxType.String() + " "
}
