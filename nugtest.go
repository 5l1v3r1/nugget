package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"./NTypes"
	"./NActions"
	"./nug2"
	"github.com/antlr/antlr4/runtime/Go/antlr"

	"reflect"
)

var (
	pathToInput string
	registers map[string]interface{}

	nodeMap map[antlr.ParseTree]interface{}

	typeRegistry map[string]reflect.Type
)

func init() {
	flag.StringVar(&pathToInput, "input", "input2.nug", "Path to input")
	flag.Parse()
	registers = make(map[string]interface{})

	//nodemap can be used to share data across constructs, just store it here whenever and retrieve based on ctx
	nodeMap = make(map[antlr.ParseTree]interface{})

	//hold a string->nugType values
	typeRegistry = make(map[string]reflect.Type)
	setupTypeRegstry()
}

func setupTypeRegstry() {
	typeRegistry["md5"] = reflect.TypeOf(NTypes.MD5{})
	typeRegistry["sha1"] = reflect.TypeOf(NTypes.SHA1{})
	typeRegistry["datetime"] = reflect.TypeOf(NTypes.Datetime{})
	typeRegistry["file"] = reflect.TypeOf(NTypes.FileInfo{})
}

func setValue(ctx antlr.ParseTree, value interface{}) {
	nodeMap[ctx] = value
}

func getValue(n antlr.ParseTree) interface{} {
	return nodeMap[n]
}

type TreeShapeListener struct {
	*parser.BaseNugget2Listener
}

func NewTreeShapeListener() *TreeShapeListener {
	return new(TreeShapeListener)
}

func (this *TreeShapeListener) EnterEveryRule(ctx antlr.ParserRuleContext) {

}

func (s *TreeShapeListener) ExitNugget_action(ctx *parser.Nugget_actionContext) {
	//grab filters if the node below us was a filter
	var myFilters []NTypes.Filter
	var action_verb string
	//test is a filter?
	if listOFilters,ok := getValue(ctx.Action_word()).([]NTypes.Filter); ok {
		//fmt.Println("ok - we have a list of filters")
		myFilters = listOFilters
		action_verb = "filter"
	} else {
		//fmt.Println(reflect.TypeOf(ctx.Action_word()))
		if av,ok := getValue(ctx.Action_word()).(string); ok {
			action_verb = av
			//fmt.Println("action verb: ", av)
		} else {
			fmt.Println("uh oh - wasn't able to determine action type")
		}
	}

	var theAction NActions.BaseAction
	switch action_verb {
	case "filter":
		//don't need to do anything here - will assign filters to actions in just a second
	case "extract":
		//todo: keyword 'files' is expected here, but don't worry about it for now
		theAction = &NActions.ExtractNTFS{}
	case "sha1":
		theAction = &NActions.SHA1Action{}
	case "md5":
		theAction = &NActions.MD5Action{}
	default:
		fmt.Println("action was not found: ", action_verb) //parser should prevent us from getting here..
	}

	if action_verb != "filter" {
		theAction.SetFilters(myFilters)
		setValue(ctx, theAction)
	}
}

func (this *TreeShapeListener) EnterDefine(ctx *parser.DefineContext) {
	isList := ctx.LISTOP() != nil
	identifier := ctx.ID().GetText()
	nugget_type := ctx.Nugget_type().GetText()

	//fmt.Println("found a define: ", ctx.ID(), " ", ctx.Nugget_type().GetText(), " is a list?: ", isList)
	if _, exists := registers[identifier]; exists {
		fmt.Println("the variable ", identifier, " already exists!")
	} else {
		switch nugget_type {
		case "ntfs":
			if isList {
				registers[identifier] = []NTypes.Extract{}
			} else {
				registers[identifier] = NTypes.Extract{}
			}
		case "file":
			if isList {
				registers[identifier] = []NTypes.FileInfo{}
			} else {
				registers[identifier] = NTypes.FileInfo{}
			}
		case "sha1":
			if isList {
				registers[identifier] = []NTypes.SHA1{}
			} else {
				registers[identifier] = NTypes.SHA1{}
			}
		case "md5"://TODO: investigate impact of 'natural types' such as string - see if we should wrap in an NType
			if isList {
				registers[identifier] = []string{}
			} else {
				registers[identifier] = ""
			}
		case "string":
			if isList {
				registers[identifier] = []NTypes.NString{}
			} else {
				registers[identifier] = NTypes.NString{}
			}
		case "pcap":
			if isList {
				registers[identifier] = []NTypes.Extract{}
			} else {
				registers[identifier] = NTypes.Extract{}
			}
		case "packet":
			if isList {
				registers[identifier] = []NTypes.NPacket{}
			} else {
				registers[identifier] = NTypes.NPacket{}
			}
		default:
			fmt.Println("Was not able to build type: ", nugget_type)
		}
	}
}

func (s *TreeShapeListener) ExitDefine_tuple(ctx *parser.Define_tupleContext) {
	// isList := ctx.LISTOP() != nil  //todo: implement lists of tuples... shudder....
	identifier := ctx.ID().GetText()

	var theTuples []interface{}

	for _, t := range ctx.AllNugget_type() {
		//fmt.Println(t.GetText())
		v := reflect.New(typeRegistry[t.GetText()])
		theTuples = append(theTuples, v)
	}

	registers[identifier] = theTuples
}

func (s *TreeShapeListener) ExitAssign(ctx *parser.AssignContext) {
	varIdentifier := ctx.ID(0).GetText()

	//if no actions, then we do a simple calculation and assign it to a register, something like: myimage = "file.dd" as ntfs
	//if len(ctx.AllNugget_action()) == 0 {
		//if it's an astype string
		if ctx.AsType() != nil {
			extractTarget := ctx.STRING().GetText()
			extractType := ctx.AsType().GetStop().GetText()
			//fmt.Println("a direct assignment has extract info: ", extractTarget, " ", extractType)
			registers[varIdentifier] = NTypes.Extract{PathToExtract: extractTarget,AsType:extractType}
	} else {
		actions := ctx.AllNugget_action()
		//setup actions if necessary
		var builtActions []NActions.BaseAction
		for _,action := range actions {
			rawAction := getValue(action)
			//if it's an extract action, we need to look behind and get some more info (like filepath and type)
			if extractAction, ok := rawAction.(*NActions.ExtractNTFS); ok {
				//todo: get real values not dummy ones
				extractAction.NTFSImageDataLocation = "G:\\school\\image\\jo.ntfs"
				extractAction.NTFSImageMetadataLocation = "/Users/myla/School/nugget/jo2.extract"
				//builtActions = append(builtActions, extractAction)
			}
			if act, ok := rawAction.(NActions.BaseAction); ok {
				builtActions = append(builtActions, act)
			}
		}

		//reverse the order of the actions
		for i := len(builtActions)/2 - 1; i >= 0; i-- {
			opp := len(builtActions) - 1 - i
			builtActions[i], builtActions[opp] = builtActions[opp], builtActions[i]
		}

		//we have raw actions, now build the chain of dependencies for each
		for index, builtAction := range builtActions {
			if index+1 < len(builtActions) {
				//fmt.Println("action at index: ", index, "is ", builtAction, " and depends on: ", builtActions[index+1])
				var depAction NActions.BaseAction
				depAction = builtActions[index+1]
				builtAction.(NActions.BaseAction).SetDependency(depAction)
			} else {
				//fmt.Println("action at index: ", index, " is ", builtAction, " and has no dependency. Setting dep to the var")
				if len(ctx.AllID()) > 1 {
					depVar := ctx.ID(1).GetText()

					// is it an existing var?
					if _, ok := registers[depVar]; ok {
						//if it's an action..
						if dep, ok := registers[depVar].(NActions.BaseAction); ok {
							//we have a datatype baseAction
							//fmt.Println("the dependency for this action will be variable: ", nVar)
							builtAction.(NActions.BaseAction).SetDependency(dep)
						}
					} else {
						fmt.Println("Error: Var '", depVar, "' not recognized.")
					}
				} else { //was not recognized, shouldn't reach here
					fmt.Println("Error: pattern not recognized.", ctx.GetText())
				}
			}
			//fmt.Println("setting the var ", varIdentifier, " to ", builtActions[0])
			registers[varIdentifier] = builtActions[0]
		}
	}
}

func (s *TreeShapeListener) ExitFilter(ctx *parser.FilterContext) {
	var allFiltersForAction []NTypes.Filter
	for i,_ := range ctx.AllFilter_term() {
		myf := getValue(ctx.Filter_term(i))
		if dep, ok := myf.(NTypes.Filter); ok {
			//fmt.Println("OH MY GOD I THINK I HAVE THIS SYSTEM FIGURED OUT ", dep)
			allFiltersForAction = append(allFiltersForAction , dep)
		}
	}
	//fmt.Println(allFiltersForAction)
	setValue(ctx, allFiltersForAction)
}

func (s *TreeShapeListener) ExitAction_word(ctx *parser.Action_wordContext) {
	if ctx.Filter() != nil {
		setValue(ctx, getValue(ctx.Filter()))
	} else {
		setValue(ctx, ctx.GetText())
	}
}

func (s *TreeShapeListener) ExitFilter_term(ctx *parser.Filter_termContext) {
	setValue(ctx, NTypes.Filter{Field: ctx.ID().GetText(), Op:ctx.COMPOP().GetText(), Value:ctx.STRING().GetText()})
}

func (s *TreeShapeListener) ExitSingleton_var(ctx *parser.Singleton_varContext) {
	theVar := ctx.ID().GetText()
	if v, ok := registers[theVar]; ok {
		fmt.Println(theVar, "[", reflect.TypeOf(v),"]:", v)
		if ba,ok := v.(NActions.BaseAction); ok {
			fmt.Println("Results for var:",theVar, ": ", ba.GetResults())
		}
	} else {
		fmt.Println("var not recognized: ", theVar)
	}
}

func (s *TreeShapeListener) ExitOperation_on_singleton(ctx *parser.Operation_on_singletonContext) {
	var operation string
	if op, ok := getValue(ctx.Singleton_op()).(string);ok {
		operation = op
	}

	theVar := ctx.ID().GetText()
	if _, ok := registers[theVar]; ok {
		switch operation {
		case "type":
			fmt.Println(reflect.TypeOf(registers[theVar]))
		case "print":
			fmt.Println(registers[theVar])
		case "size":
			fmt.Println("len not implemented yet")
		default:
			fmt.Println("operation not recognized..")
		}
	} else {
		fmt.Println("var not recognized:", theVar)
	}
}

func (s *TreeShapeListener) ExitSingleton_op(ctx *parser.Singleton_opContext) {
	setValue(ctx, ctx.GetText())
}

func main() {
	file, err := os.Open(pathToInput)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		input := antlr.NewInputStream(scanner.Text())
		lexer := parser.NewNugget2Lexer(input)
		stream := antlr.NewCommonTokenStream(lexer, 0)
		p := parser.NewNugget2Parser(stream)
		p.AddErrorListener(antlr.NewDiagnosticErrorListener(true))
		p.BuildParseTrees = true
		tree := p.Prog()
		antlr.ParseTreeWalkerDefault.Walk(NewTreeShapeListener(), tree)
	}
}
