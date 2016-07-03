package main

import (
	"fmt"
	"math/rand"
	"os"
)

/**
 * Send your busters out into the fog to trap ghosts and bring them home!
 **/

type eState int

const (
	IDLE eState = iota
	EXPLORE
	TRACK
	CAPTURE
	RETURN
	RELEASE
	STUN
	STUNNED
	POP
)

type (
	State interface {
		enter(b *Buster)
		update(b *Buster) eState
		exit(b *Buster)

		id() eState

		//output()
	}

	Vector struct {
		x int
		y int
	}

	IdleState    struct{}
	ExploreState struct {
		path Path
	}
	TrackState   struct{}
	CaptureState struct{}
	ReturnState  struct {
		path Path
	}
	ReleaseState struct{}
	StunState    struct{}
	StunnedState struct {
		stunTurn int
	}

	FSM struct {
		states []State
		stack  []State
		buster *Buster
	}

	BasicFSM struct {
		FSM
	}

	Waypoint struct {
		w Vector
		r int
	}

	Path struct {
		waypoints []Waypoint
		index     int
		done      bool
		repeat    bool
	}

	Buster struct {
		position Vector
		target   Vector
		targetId int
		stunTurn int
		seen     bool
		fsm      *FSM
	}

	Ghost struct {
		position Vector
		lastSeen int
		captured bool
	}
)

var (
	busters          []Buster
	ghosts           []Ghost
	myTeamId         int
	bustersPerPlayer int
	ghostCount       int
	seenGhosts       []int
	seenGhostCount   int

	seenBusters      []int
	seenBustersCount int

	captured int
)

//=========================== Vector ==============================================

func (v Vector) distanceSqr(target Vector) int {
	x := v.x - target.x
	y := v.y - target.y

	return x*x + y*y
}

//=========================== States ==============================================

// Idle

func (s *IdleState) enter(b *Buster) {
}

func (s *IdleState) update(b *Buster) eState {
	return EXPLORE
}

func (s *IdleState) exit(b *Buster) {
}

func (s *IdleState) id() eState {
	return IDLE
}

// Explore

func (s *ExploreState) enter(b *Buster) {
	s.path.update(b.position)
	s.path.print()

	if s.path.count() > 0 {
		b.target = s.path.nextWaypoint().w
	} else {
		b.target = Vector{x: rand.Intn(16000), y: rand.Intn(9000)}
	}
}

func (s *ExploreState) update(b *Buster) eState {
	s.path.update(b.position)
	s.path.print()

	if s.path.count() > 0 {
		// if s.path.done {
		//     if b.position.distanceSqr(b.target) < 100 {
		//         b.target = Vector{x:rand.Intn(16000), y:rand.Intn(9000)}
		//     }
		// } else {
		b.target = s.path.nextWaypoint().w
		// }
	} else {

		if b.position.distanceSqr(b.target) < 100 {
			b.target = Vector{x: rand.Intn(16000), y: rand.Intn(9000)}
		}
	}

	return EXPLORE
}

func (s *ExploreState) exit(b *Buster) {
}

func (s *ExploreState) id() eState {
	return EXPLORE
}

// Track
func (s *TrackState) enter(b *Buster) {
}

func (s *TrackState) update(b *Buster) eState {
	g := ghosts[b.targetId]
	b.target = g.position

	var distance = b.position.distanceSqr(b.target)

	if distance > 810000 && distance < 3097600 {
		return CAPTURE
	}

	if g.lastSeen != 0 {
		return POP
	}

	return TRACK
}

func (s *TrackState) exit(b *Buster) {
}

func (s *TrackState) id() eState {
	return TRACK
}

// Capture
func (s *CaptureState) enter(b *Buster) {
}

func (s *CaptureState) update(b *Buster) eState {
	// wait for it
	g := ghosts[b.targetId]
	if g.lastSeen != 0 {
		return POP
	}
	return CAPTURE
}

func (s *CaptureState) exit(b *Buster) {
}

func (s *CaptureState) id() eState {
	return CAPTURE
}

// Return
func (s *ReturnState) enter(b *Buster) {
	if myTeamId == 0 {
		b.target = Vector{x: 0, y: 0}
	} else {
		b.target = Vector{x: 16000, y: 9000}
	}

	s.path = Path{waypoints: make([]Waypoint, 0, 5), done: false}
	s.path.pushWaypoint(b.target, 1600)
}

func (s *ReturnState) update(b *Buster) eState {

	// if seen enemy nearby, stay out of range while still trying to get back home...

	s.path.update(b.position)
	s.path.print()

	if s.path.done {
		return RELEASE
	} else {
		b.target = s.path.nextWaypoint().w
	}

	return RETURN
}

func (s *ReturnState) exit(b *Buster) {
}

func (s *ReturnState) id() eState {
	return RETURN
}

// Release
func (s *ReleaseState) enter(b *Buster) {
}

func (s *ReleaseState) update(b *Buster) eState {
	b.targetId = -1
	return POP
}

func (s *ReleaseState) exit(b *Buster) {
	captured++
}

func (s *ReleaseState) id() eState {
	return RELEASE
}

// Stun
func (s *StunState) enter(b *Buster) {
}

func (s *StunState) update(b *Buster) eState {
	if busters[b.targetId].fsm.state() == STUNNED || busters[b.targetId].seen == false {
		return POP
	}

	return STUN
}

func (s *StunState) exit(b *Buster) {
	b.stunTurn = 20
}

func (s *StunState) id() eState {
	return STUN
}

// Stunned
func (s *StunnedState) enter(b *Buster) {
	s.stunTurn = 10
}

func (s *StunnedState) update(b *Buster) eState {
	s.stunTurn--
	if s.stunTurn == 0 {
		return POP
	}

	return STUNNED
}

func (s *StunnedState) exit(b *Buster) {
}

func (s *StunnedState) id() eState {
	return STUNNED
}

//=========================== FSM =================================================

func NewBasicFSM() *FSM {
	return &FSM{
		states: []State{&IdleState{}, &ExploreState{}, &TrackState{}, &CaptureState{}, &ReturnState{}, &ReleaseState{}, &StunState{}, &StunnedState{}},
		stack:  make([]State, 10),
	}
}

func (fsm *FSM) peek() State {
	return fsm.stack[len(fsm.stack)-1]
}

func (fsm *FSM) push(state eState) {
	fsm.stack = append(fsm.stack, fsm.states[state])
	fsm.peek().enter(fsm.buster)
}

func (fsm *FSM) pop() eState {
	state := fsm.peek()
	state.exit(fsm.buster)

	fsm.stack = fsm.stack[0 : len(fsm.stack)-1]
	return state.id()
}

func (fsm *FSM) state() eState {
	return fsm.peek().id()
}

func (fsm *FSM) update(b *Buster) {
	state := fsm.peek().update(b)
	if state != fsm.state() {
		fsm.pop()
		if state != POP {
			fsm.push(state)
		} else {
			fsm.peek().enter(fsm.buster)
		}
	}
}

func (fsm *FSM) print() {
	fmt.Fprintf(os.Stderr, "FSM ")
	for _, v := range fsm.stack {
		if v != nil {
			fmt.Fprintf(os.Stderr, "[%v]", v.id())
		}
	}
	fmt.Fprintf(os.Stderr, "\n")
}

//=========================== Path ================================================

func (p *Path) pushWaypoint(w Vector, r int) {
	p.waypoints = append(p.waypoints, Waypoint{w: w, r: r})
	p.done = false
}

func (p *Path) popWaypoint() Waypoint {
	w := p.waypoints[p.count()-1]
	p.waypoints = p.waypoints[0 : p.count()-1]
	return w
}

func (p *Path) nextWaypoint() Waypoint {
	return p.waypoints[p.index]
}

func (p *Path) update(position Vector) {
	if p.count() > 0 {
		nextWaypoint := p.nextWaypoint()
		fmt.Fprintf(os.Stderr, "update %v- %v\n", nextWaypoint, position)
		if nextWaypoint.w.distanceSqr(position) <= nextWaypoint.r*nextWaypoint.r {
			p.index++
		}

		if p.repeat {
			if p.count() == p.index {
				p.index = 0
			}
		} else {
			if p.count() == p.index {
				p.done = true
			}
		}
	}
}

func (p *Path) count() int {
	return len(p.waypoints)
}

func (p *Path) print() {
	fmt.Fprintf(os.Stderr, "Path %v\n", p)
}

//=========================== Buster ==============================================
func (b *Buster) hasGhost(value bool) {
	if value {
		b.fsm.pop()
		b.fsm.push(RETURN)
	}
}

func (b *Buster) track(id int) bool {
	if id == -1 {
		return false
	}

	if b.fsm.state() == IDLE || b.fsm.state() == EXPLORE {
		b.targetId = id
		b.target = ghosts[id].position
		b.fsm.push(TRACK)

		return true
	}

	return false
}

func (b *Buster) stun(busterId int) {
	b.targetId = busterId
	b.fsm.push(STUN)
}

func (b *Buster) initialise() {
	b.fsm = NewBasicFSM()
	b.fsm.buster = b
	b.stunTurn = 0
}

func (b *Buster) update() {
	b.fsm.update(b)
}

func (b *Buster) output() {
	// TODO Move this into the individual states
	switch b.fsm.state() {
	case IDLE:
		{
			fmt.Printf("MOVE %d %d IDLE\n", b.position.x, b.position.y)
		}

	case EXPLORE:
		{
			fmt.Printf("MOVE %d %d EXPLORE\n", b.target.x, b.target.y)
		}

	case TRACK:
		{
			var distance = b.position.distanceSqr(b.target)
			fmt.Printf("MOVE %d %d TRACK Id %d - %d\n", b.target.x, b.target.y, b.targetId, distance)
		}

	case CAPTURE:
		{
			fmt.Printf("BUST %d\n", b.targetId)
		}

	case RETURN:
		{
			fmt.Printf("MOVE %d %d Returning ... \n", b.target.x, b.target.y)
		}

	case RELEASE:
		{
			fmt.Printf("RELEASE\n")
		}

	case STUN:
		{
			fmt.Printf("STUN %d Stunning\n", b.targetId)
		}

	case STUNNED:
		{
			fmt.Printf("MOVE 0 0 STUNNED\n")
		}
	}
}

func main() {
	// bustersPerPlayer: the amount of busters you control
	fmt.Scan(&bustersPerPlayer)

	busters = make([]Buster, bustersPerPlayer*2)

	// ghostCount: the amount of ghosts on the map
	fmt.Scan(&ghostCount)

	ghosts = make([]Ghost, ghostCount)
	for i := 0; i < ghostCount; i++ {
		ghosts[i].lastSeen = -1
	}

	seenBusters = make([]int, bustersPerPlayer)
	seenGhosts = make([]int, ghostCount)

	// myTeamId: if this is 0, your base is on the top left of the map, if it is one, on the bottom right
	fmt.Scan(&myTeamId)

	// initialise busters
	for i := 0; i < 2*bustersPerPlayer; i++ {
		busters[i].initialise()
		busters[i].fsm.push(EXPLORE)
		var state *ExploreState = (busters[i].fsm.states[EXPLORE]).(*ExploreState)
		if i%bustersPerPlayer == 0 {
			state.path = Path{waypoints: make([]Waypoint, 0, 5), done: false}
			state.path.pushWaypoint(Vector{x: 13801, y: 2200}, 50)
			state.path.pushWaypoint(Vector{x: 13801, y: 6801}, 50)
			state.path.pushWaypoint(Vector{x: 2200, y: 6801}, 50)
			state.path.repeat = true
		} else if (i+1)%bustersPerPlayer == 0 {
			state.path = Path{waypoints: make([]Waypoint, 0, 5), done: false}
			state.path.pushWaypoint(Vector{x: 2200, y: 6801}, 50)
			state.path.pushWaypoint(Vector{x: 13801, y: 6801}, 50)
			state.path.pushWaypoint(Vector{x: 8000, y: 4500}, 50)
			state.path.repeat = true
		}
	}

	for {
		// entities: the number of busters and ghosts visible to you
		var entities int
		fmt.Scan(&entities)

		for j := 0; j < seenBustersCount; j++ {
			ob := &busters[seenBusters[j]]
			ob.seen = false
		}

		seenGhostCount = 0
		seenBustersCount = 0

		for i := 0; i < ghostCount; i++ {
			if ghosts[i].lastSeen != -1 {
				ghosts[i].lastSeen++
			}
		}

		for i := 0; i < entities; i++ {
			// entityId: buster id or ghost id
			// y: position of this buster / ghost
			// entityType: the team id if it is a buster, -1 if it is a ghost.
			// state: For busters: 0=idle, 1=carrying a ghost.
			// value: For busters: Ghost id being carried. For ghosts: number of busters attempting to trap this ghost.
			var entityId, x, y, entityType, state, value int
			fmt.Scan(&entityId, &x, &y, &entityType, &state, &value)

			switch entityType {
			case -1:
				{
					ghosts[entityId].position.x = x
					ghosts[entityId].position.y = y
					ghosts[entityId].lastSeen = 0

					seenGhosts[seenGhostCount] = entityId
					seenGhostCount++

					fmt.Fprintf(os.Stderr, "GHOST: %d SEEN %d at %d %d\n", seenGhostCount, entityId, x, y)
				}
			default:
				{
					busters[entityId].position.x = x
					busters[entityId].position.y = y

					if entityId >= myTeamId*bustersPerPlayer && entityId < bustersPerPlayer*(myTeamId+1) {
						busters[entityId].hasGhost(state == 1)
						if busters[entityId].fsm.state() != STUNNED && state == 2 {
							busters[entityId].fsm.push(STUNNED)
						}

						if busters[entityId].fsm.state() == STUNNED && state != 2 {
							busters[entityId].fsm.pop()
						}

						if busters[entityId].fsm.state() == RETURN && state != 1 {
							busters[entityId].fsm.pop()
						}

					} else {
						busters[entityId].seen = true

						busters[entityId].fsm.pop()

						switch state {
						case 0:
							busters[entityId].fsm.push(EXPLORE)
						case 1:
							busters[entityId].fsm.push(RETURN)
						case 2:
							busters[entityId].fsm.push(STUNNED)
						case 3:
							busters[entityId].fsm.push(CAPTURE)
						}

						seenBusters[seenBustersCount] = entityId
						seenBustersCount++

						fmt.Fprintf(os.Stderr, "ENEMY: %d SEEN %d at %d %d\n", seenBustersCount, entityId, x, y)
					}

					if busters[entityId].stunTurn > 0 {
						busters[entityId].stunTurn--
					}
				}
			}
		}

		stalk()
		btog()
		stun()

		for i := myTeamId * bustersPerPlayer; i < myTeamId*bustersPerPlayer+bustersPerPlayer; i++ {
			busters[i].update()
			busters[i].output()
			busters[i].fsm.print()
		}
	}
}

func btog() {
	var lastSeen int = 5
	var lastSeenIndex int = -1
	if seenGhostCount == 0 {
		for i, g := range ghosts {
			if g.lastSeen < lastSeen && g.lastSeen < 5 {
				lastSeen = g.lastSeen
				lastSeenIndex = i
			}
		}
	}

	for i := myTeamId * bustersPerPlayer; i < myTeamId*bustersPerPlayer+bustersPerPlayer; i++ {
		b := &busters[i]

		if b.fsm.state() != STUNNED {
			if seenGhostCount > 0 {
				var closest int = -1
				var d = 999999999
				for j := 0; j < seenGhostCount; j++ {
					g := ghosts[seenGhosts[j]]
					dsqr := b.position.distanceSqr(g.position)
					if dsqr < d {
						closest = seenGhosts[j]
						d = dsqr
					}
				}

				b.track(closest)
			} else {
				b.track(lastSeenIndex)
			}
		}
	}
}

func stun() {
	for i := myTeamId * bustersPerPlayer; i < (myTeamId+1)*bustersPerPlayer; i++ {
		b := &busters[i]

		if b.stunTurn == 0 && b.fsm.state() != STUNNED {
			for j := 0; j < seenBustersCount; j++ {
				ob := &busters[seenBusters[j]]

				dsqr := b.position.distanceSqr(ob.position)

				//fmt.Fprintf(os.Stderr, "dist %d\n", dsqr)

				if dsqr <= 3097600 && ob.fsm.state() != STUNNED && ob.fsm.state() == RETURN {
					b.stun(seenBusters[j])
					break
				}
			}
		}
	}
}

func stalk() {

	// if captured >= ghostCount / 2 {
	//     for i := myTeamId * bustersPerPlayer; i < myTeamId * bustersPerPlayer + bustersPerPlayer; i++ {
	//         b := &busters[i]

	//         if b.fsm.state() != STUNNED {
	//             if seenBustersCount > 0 {
	//                 var closest int = -1
	//                 var d = 999999999
	//                 for j := 0; j < seenBustersCount; j++ {
	//                     ob := &busters[seenBusters[j]]

	//                     dsqr := b.position.distanceSqr(ob.position)
	//                     if dsqr < d {
	//                         closest = seenGhosts[j]
	//                         d = dsqr
	//                     }
	//                 }

	//                 if dsqr <= 3097600 && ob.fsm.state() != STUNNED && ob.fsm.state() == RETURN {
	//                     b.stalk(seenBusters[j])
	//                     break
	//                 }
	//             }
	//         }
	//     }
	// }
}
