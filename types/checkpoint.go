package types

import (
	"log"
	"math"

	"github.com/hectagon-finance/chain-mvp/third_party/tree"
	"github.com/hectagon-finance/chain-mvp/third_party/utils"
)

const NoFallbackOption = math.MaxUint64 - 1
const EndOfMission = math.MaxUint64 - 2
const GenesisBlock = 0

type CheckPoint struct {
	Id               string
	Title            string
	Description      string
	FallbackId       uint64
	children         []*CheckPoint
	voteMachine      VotingMachine
	lastBlockToVote  uint64
	lastBlockToTally uint64
	blockchain       Blockchain
}

// return something that is printable
func (n *CheckPoint) Data() interface{} {
	return n.Title
}

// cannot return n.children directly.
// https://github.com/golang/go/wiki/InterfaceSlice
func (n *CheckPoint) Children() (c []tree.Node) {
	for _, child := range n.children {
		c = append(c, tree.Node(child))
	}
	return
}

func CreateEmptyCheckPoint(title string, desc string, b VotingMachine, blockchain Blockchain) *CheckPoint {
	CheckPoint := CheckPoint{
		Id:               utils.RandString(16),
		Title:            title,
		Description:      desc,
		children:         []*CheckPoint{},
		voteMachine:      b,
		FallbackId:       NoFallbackOption,
		lastBlockToVote:  GenesisBlock,
		lastBlockToTally: GenesisBlock,
		blockchain:       blockchain,
	}
	return &CheckPoint
}
func CreateCheckPoinWithChildren(name string, desc string, children []*CheckPoint, b VotingMachine, fallbackId uint64, lastBlockToVote uint64, lastBlockToTally uint64, blockchain Blockchain) *CheckPoint {
	c := CheckPoint{
		Id:               utils.RandString(16),
		Title:            name,
		Description:      desc,
		FallbackId:       fallbackId,
		children:         children,
		voteMachine:      b,
		lastBlockToVote:  lastBlockToVote,
		lastBlockToTally: lastBlockToTally,
		blockchain:       blockchain,
	}
	return &c
}
func (this *CheckPoint) Attach(child *CheckPoint) *CheckPoint {
	if this.children == nil {
		this.children = make([]*CheckPoint, 0)
	}
	this.children = append(this.children, child)
	return child
}

func (this *CheckPoint) SetVotingMachine(v VotingMachine) {
	this.voteMachine = v
}

/**
* Conversational text the describe the current state of the CheckPoint
* including: Title, Description, Options, How voting will conduct
**/
func (this *CheckPoint) Print() {
	log.Printf("%s\n%s\nVoting Mechanism:\n%s\n", this.Title, this.Description, this.voteMachine.Desc())
	for i := range this.children {
		log.Printf("- opt %d: %s\n", i, this.children[i].Title)
	}
	log.Printf("\n")
}
func (this *CheckPoint) Get(idx uint64) *CheckPoint {
	if idx < uint64(len(this.children)) {
		return this.children[idx]
	}
	return nil
}
func (this *CheckPoint) start(lastTalliedResult []byte) bool {
	if this.children == nil {
		return false
	}
	if len(this.children) == 0 || this.FallbackId == NoFallbackOption {
		return false
	}
	return this.voteMachine.Start(lastTalliedResult, uint64(len(this.children)), this.FallbackId)
}
func (this *CheckPoint) isValidChoice(option []byte) bool {
	if this.voteMachine.IsStarted() == false {
		return false
	}
	return this.voteMachine.ValidateVote(option)
}

/**
* Function vote
* Params: tr *Mission, who string, input []byte
* Returns: recordStatus ExecutionStatus, tallyStatus ExecutionStatus, newChkPStatus ExecutionStatus, fallbackAttempt bool
* TODO: what if we want to hide the voter's option from validator?
 */
func (this *CheckPoint) vote(tr *Mission, who string, input []byte) (ExecutionStatus, ExecutionStatus, ExecutionStatus, bool) {
	var recordStatus ExecutionStatus = DIDNOTSTART
	var tallyStatus ExecutionStatus = DIDNOTSTART
	var newChkPStatus ExecutionStatus = DIDNOTSTART
	fallbackAttempt := false
	if this.voteMachine.Record(who, input) == true {
		recordStatus = SUCCEED
	} else {
		recordStatus = FAILED
	}
	// check for fallback
	fallbackAttempt, newChkPStatus = fallback(tr, this.voteMachine, this.FallbackId)
	// then check for tally
	if fallbackAttempt == false && this.voteMachine.ShouldTally() == true {
		tallyStatus, newChkPStatus = tally(tr, this.voteMachine)
	}
	return recordStatus, tallyStatus, newChkPStatus, fallbackAttempt
}

/**
* Func tally; count all the vote
* Args: tr *Mission, m VotingMachine, input []byte
* Return: tallyStatus ExecutionStatus, newChkPoinStatus ExecutionStatus
 */
func tally(tr *Mission, m VotingMachine) (ExecutionStatus, ExecutionStatus) {
	_tallyStatus, _, tallyResult, selectedOption := m.Tally()
	_newChkPointStatus := false
	tallyStatus := FAILED
	newChkPointStatus := DIDNOTSTART
	if _tallyStatus == true {
		tallyStatus = SUCCEED
		if selectedOption != NoOptionMade {
			_newChkPointStatus, _ = tr.Choose(selectedOption, tallyResult)
			if _newChkPointStatus == true {
				newChkPointStatus = SUCCEED
			} else {
				newChkPointStatus = FAILED
			}
		}
	}
	return tallyStatus, newChkPointStatus
}

/**
* Func fallback; check if Voter can no longer vote, Mission can no longer tally then choose fallbackId
* Args: tr *Mission, m VotingMachine, fallbackId uint64, input []byte
* Return: fallbackAttempt bool, newChkPointStatus ExecutionStatus
 */
func fallback(tr *Mission, m VotingMachine, fallbackId uint64) (bool, ExecutionStatus) {
	currentBlk := tr.currentChkP.blockchain.GetCurrentBlockNumber()
	lastBlkVote := tr.currentChkP.lastBlockToVote
	lastBlkTally := tr.currentChkP.lastBlockToTally
	tallyResult, selectedOption := m.GetTallyResult()
	newChkPointStatus := DIDNOTSTART
	if currentBlk > lastBlkVote && currentBlk > lastBlkTally && selectedOption == NoOptionMade {
		_newChkPointStatus, _ := tr.Choose(fallbackId, tallyResult)
		if _newChkPointStatus == true {
			newChkPointStatus = SUCCEED
		} else {
			newChkPointStatus = FAILED
		}
		return true, newChkPointStatus
	}
	return false, newChkPointStatus
}
