package game

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/wok/rent-a-hero/internal/engine"
)

const loc04Actor = "RodrigoLarge"
const loc04CustomerActor = "RamWalk"

const (
	FlagLoc04CustomerState = flagLoc05QuestA
	FlagLoc04VisitCount    = "loc04.visit_count"
	FlagLoc04PaymentOnDesk = "loc04.payment_on_desk"
	FlagLoc04CustomerSeen  = "loc04.customer_seen"
	FlagLoc04CustomerStory = "loc04.customer_story"
	loc04CustomerPresent   = "loc04.customer_present"
	loc04DesktopToggle     = "loc04.desktop_toggle"
	loc04EberToggle        = "loc04.eber_toggle"
	loc04ElefantToggle     = "loc04.elefant_toggle"
	loc04CustomerStage     = "loc04.customer_stage"
	loc04Seated            = "loc04.seated"
)

var loc04DormantLayers = []string{
	"RodTalk", "RodDeskTalk", "RodSetzenLayer", "RodTake", "RodRein", "RodRaus", "RodSetzen",
	"RamTalk", "RodSitAct1", "RodSitAct2", "RamilRein", "RamilRaus", "RamilGeld",
}

var loc04CustomerAreas = []string{"006_DoorRamil", "006_NewsRamil", "006_FassRamil", "RamTalk"}
var loc04BaseAreas = []string{"006_Desktop", "006_Eber", "006_Kiste", "006_Elefant", "006_Kasten", "006_Door", "006_Fass", "006_News", "006_Lade2", "006_Lade3", "006_Lade4"}

type loc04Runtime struct {
	baseBackground    *engine.Background
	rodTalkBackground *engine.Background
	ramTalkBackground *engine.Background
}

type LOC04Controller struct{}

func (LOC04Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	if scene != "006" {
		return nil
	}
	previousVisits := ctx.GetFlag(FlagLoc04VisitCount)
	customerState := ctx.GetFlag(FlagLoc04CustomerState)
	loc04PrepareBackgrounds(ctx, customerState == 1 || previousVisits == 0, customerState == 1)
	tasks := []engine.Task{
		ctx.PlayMusic("Loc04_RodrigosOffice.wav"),
		loc04InitializeScene(ctx),
		ctx.SetFlag(FlagLoc04VisitCount, previousVisits+1),
		ctx.SetFlag(loc04DesktopToggle, 0),
		ctx.SetFlag(loc04CustomerPresent, 0),
		ctx.SetFlag(loc04EberToggle, 0),
		ctx.SetFlag(loc04ElefantToggle, 0),
		ctx.SetFlag(loc04Seated, 0),
		loc04EnterOffice(ctx),
	}
	if customerState == 1 {
		tasks = append(tasks, loc04SitForCustomer(ctx), loc04CustomerArrival(ctx))
	} else if previousVisits == 0 {
		tasks = append(tasks, loc04FirstVisit(ctx))
	}
	return engine.Sequence(tasks...)
}

func (LOC04Controller) Exit(ctx *Context, scene string, to string) engine.Task { return nil }

func (LOC04Controller) Click(ctx *Context, area string) engine.Task {
	switch area {
	case "006_Desktop":
		if ctx.GetFlag(FlagLoc04PaymentOnDesk) != 0 {
			return engine.Sequence(
				loc04StandIfNeeded(ctx),
				loc04OfficeWalkFacing(ctx, loc04Actor, 0x83, 0xF6, 6),
				ctx.HideActor(loc04Actor),
				loc04PlayRange(ctx, "RodTake", 0, 9),
				ctx.PlaySFX("Sfx_Kramen.wav"),
				loc04PlayRange(ctx, "RodTake", 10, -1),
				ctx.ShowActor(loc04Actor),
				loc04Speak(ctx, loc04Actor, "006_ROD_05"),
				ctx.SetFlag(FlagLoc04PaymentOnDesk, 0),
				ctx.AddItem(7),
			)
		}
		line := "006_ROD_03"
		if ctx.GetFlag(loc04DesktopToggle) != 0 {
			line = "006_ROD_04"
		}
		return engine.Sequence(loc04ContextSpeech(ctx, 0x10E, 0x120, 3, line), ctx.SetFlag(loc04DesktopToggle, 1-ctx.GetFlag(loc04DesktopToggle)))
	case "006_Eber":
		line := "006_ROD_06"
		if ctx.GetFlag(loc04EberToggle) != 0 {
			line = "006_ROD_07"
		}
		return engine.Sequence(loc04ContextSpeech(ctx, 0x150, 0x100, 4, line), ctx.SetFlag(loc04EberToggle, 1-ctx.GetFlag(loc04EberToggle)))
	case "006_Kiste":
		return loc04ContextSpeech(ctx, 0x167, 0x104, 5, "006_ROD_10")
	case "006_Elefant":
		line := "006_ROD_08"
		if ctx.GetFlag(loc04ElefantToggle) != 0 {
			line = "006_ROD_09"
		}
		return engine.Sequence(loc04ContextSpeech(ctx, 0x96, 0x158, 3, line), ctx.SetFlag(loc04ElefantToggle, 1-ctx.GetFlag(loc04ElefantToggle)))
	case "006_Kasten":
		return engine.Sequence(loc04StandIfNeeded(ctx), loc04SpeakAt(ctx, 0x19D, 0x129, 5, "006_ROD_11"))
	case "006_Door":
		return engine.Sequence(loc04StandIfNeeded(ctx), loc04OfficeWalkFacing(ctx, loc04Actor, 300, 350, 7), &loc04DoorWalkOutTask{ctx: ctx}, ctx.HideActor(loc04Actor), ctx.ShowLayer("RodRaus"), ctx.PlayLayerFrames("RodRaus", 0, 6), ctx.PlaySFX("Sfx_Door_CreakShort.wav"), ctx.PlayLayerFrames("RodRaus", 7, 0x12), ctx.PlaySFX("Sfx_Tock.wav"), ctx.PlayLayerFrames("RodRaus", 0x13, -1), ctx.HideLayer("RodRaus"), ctx.ChangeLocation(1, loc7ASceneID))
	case "006_DoorRamil":
		return engine.Sequence(loc04PoseCustomerAct1(ctx), ctx.PlayLayerFrames("RodSitAct1", 0x13, 5), loc04SpeakLayer(ctx, "RamTalk", "006_RAM_02", 0, -1, true), ctx.PlayLayerFrames("RodSitAct1", 5, 0x13), loc04PoseCustomer(ctx), loc04SpeakRodMode(ctx, "006_ROD_12", 2))
	case "006_NewsRamil":
		return loc04SpeakRodMode(ctx, "006_ROD_27", 2)
	case "006_FassRamil":
		return loc04SpeakRodMode(ctx, "006_ROD_26", 2)
	case "RamTalk", "006_ramtalk":
		return loc04AdvanceCustomerConversation(ctx)
	case "006_Fass":
		return loc04ContextSpeech(ctx, 0x10E, 0x120, 3, "006_ROD_26")
	case "006_News":
		return loc04ContextSpeech(ctx, 0x10E, 0x120, 3, "006_ROD_27")
	case "006_Lade2":
		return engine.Sequence(loc04StandIfNeeded(ctx), loc04SpeakAt(ctx, 0x19D, 0x129, 5, "006_ROD_28"))
	case "006_Lade3":
		return engine.Sequence(loc04StandIfNeeded(ctx), loc04SpeakAt(ctx, 0x19D, 0x129, 5, "006_ROD_29"))
	case "006_Lade4":
		return engine.Sequence(loc04StandIfNeeded(ctx), loc04SpeakAt(ctx, 0x19D, 0x129, 5, "006_ROD_30"))
	}
	return nil
}

func (LOC04Controller) UseItem(ctx *Context, item int, area string) engine.Task { return nil }
func (LOC04Controller) SelectItem(ctx *Context, item int) engine.Task           { return nil }

func (LOC04Controller) LoadConditionMask(ctx *Context, scene string) int {
	if scene != "006" {
		return 0
	}
	mask := 0
	if ctx.GetFlag(FlagLoc04CustomerState) == 1 {
		mask = 3
	}
	if ctx.GetFlag(FlagLoc04PaymentOnDesk) != 0 {
		mask |= 2
	}
	return mask
}

func loc04InitializeScene(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		scene := ctx.session.scene
		for _, id := range []string{"RodRein", "RodRaus", "RodSetzen", "RamilRein", "RamilRaus", "RamilGeld"} {
			if layer, ok := scene.Layers[id]; ok {
				layer.Presentation = true
				layer.ColorKeyed = false
			}
		}
		loc04AuditLayers(ctx)
		for _, id := range loc04DormantLayers {
			if layer, ok := scene.Layers[id]; ok {
				layer.Visible = false
				layer.Playing = false
				layer.Frame = 0
				layer.Accumulator = 0
			}
		}
		if layer, ok := scene.Layers["RamTalk"]; ok && layer.Source != nil {
			if _, exists := scene.Areas["RamTalk"]; !exists {
				w := layer.Source.Width() * layer.Zoom / 100
				h := layer.Source.Height() * layer.Zoom / 100
				area := &engine.Area{ID: "RamTalk", X1: layer.X, Y1: layer.Y, X2: layer.X + w, Y2: layer.Y + h, CursorType: 10}
				ctx.session.localizeArea(area, ctx.session.state.Location)
				scene.Areas["RamTalk"] = area
				scene.AreaOrder = append(scene.AreaOrder, "RamTalk")
			}
		}
		for _, id := range loc04CustomerAreas {
			if a, ok := scene.Areas[id]; ok {
				a.Enabled = false
			}
		}
		for _, id := range loc04BaseAreas {
			if a, ok := scene.Areas[id]; ok {
				a.Enabled = true
			}
		}
		if player, playerID, ok := ctx.session.actorByNameOrAsset(loc04Actor); ok {
			for id, actor := range scene.Characters {
				if actor == nil || actor == player {
					continue
				}
				stem := strings.TrimSuffix(filepath.Base(actor.AssetName), filepath.Ext(actor.AssetName))
				if strings.HasPrefix(strings.ToLower(id), "rodrigo") || strings.HasPrefix(strings.ToLower(stem), "rodrigo") {
					actor.Visible = false
				}
			}
			player.Visible = true
			for i, id := range scene.CharacterOrder {
				if id == playerID {
					copy(scene.CharacterOrder[1:i+1], scene.CharacterOrder[0:i])
					scene.CharacterOrder[0] = playerID
					break
				}
			}
			w, h := 0, 0
			if player.Source != nil {
				w, h = player.Source.Width(), player.Source.Height()
			}
			log.Printf(`game: scene "006": using office actor %q from %q (%dx%d, zoom=%d, z=%d, direction=%d)`, playerID, player.AssetName, w, h, player.Zoom, player.ZPos, player.Direction)
			for _, id := range scene.CharacterOrder {
				actor := scene.Characters[id]
				if actor == nil {
					continue
				}
				aw, ah := 0, 0
				if actor.Source != nil {
					aw, ah = actor.Source.Width(), actor.Source.Height()
				}
				log.Printf(`game: scene "006": character %q asset=%q size=%dx%d zoom=%d z=%d direction=%d visible=%t`, id, actor.AssetName, aw, ah, actor.Zoom, actor.ZPos, actor.Direction, actor.Visible)
			}
		} else {
			log.Printf(`game: scene "006": required office actor asset %q is missing`, loc04Actor)
		}
		if a, _, ok := ctx.session.actorByNameOrAsset(loc04CustomerActor); ok {
			a.Visible = false
		}
	})
}

func loc04EnterOffice(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.HideActor(loc04Actor),
		loc04PlayRange(ctx, "RodRein", 0, 0x11),
		ctx.PlaySFX("Sfx_Tock.wav"),
		loc04PlayRange(ctx, "RodRein", 0x12, -1),
		&loc04DoorWalkInTask{ctx: ctx},
		loc04PoseStanding(ctx),
	)
}

const loc04EntranceTransformHz = 25.0

type loc04DoorWalkInTask struct {
	ctx            *Context
	actor          *engine.Actor
	phase          int
	frameClock     float64
	transformClock float64
	transformN     int
}

func (t *loc04DoorWalkInTask) Update(dt float64) bool {
	if t.actor == nil {
		actor, _, ok := t.ctx.session.actorByNameOrAsset(loc04Actor)
		if !ok || actor == nil {
			log.Printf(`game: scene "006": entrance transform: required actor %q is missing`, loc04Actor)
			return true
		}
		t.actor = actor
		t.actor.Visible = true
		t.actor.Zoom = 115
		t.setFromTransform(442, 109, 115)
		t.actor.Frame = 99
	}

	if t.phase == 0 {
		t.frameClock += dt
		for t.frameClock >= 1.0/engine.DefaultWalkAnimFPS {
			t.frameClock -= 1.0 / engine.DefaultWalkAnimFPS
			if t.actor.Frame < 101 {
				t.actor.Frame++
			} else {
				t.phase = 1
				t.actor.Frame = 48
				break
			}
		}
		return false
	}

	t.transformClock += dt
	for t.transformClock >= 1.0/loc04EntranceTransformHz && t.transformN < 30 {
		t.transformClock -= 1.0 / loc04EntranceTransformHz
		t.transformN++
		p := float64(t.transformN) / 30.0
		x := 442.0 + (223.0-442.0)*p
		y := 109.0 + (122.0-109.0)*p
		zoom := 115.0 + (72.0-115.0)*p
		t.setFromTransform(x, y, zoom)
		t.actor.Zoom = int(zoom + 0.5)
	}
	t.frameClock += dt
	for t.frameClock >= 1.0/engine.DefaultWalkAnimFPS {
		t.frameClock -= 1.0 / engine.DefaultWalkAnimFPS
		t.actor.Frame++
		if t.actor.Frame > 59 {
			t.actor.Frame = 48
		}
	}
	if t.transformN < 30 {
		return false
	}
	t.actor.X = 300
	t.actor.Y = 350
	t.actor.Zoom = 72
	return true
}

func (t *loc04DoorWalkInTask) setFromTransform(x, y, zoom float64) {
	anchorX, anchorY := t.actor.AnchorX, t.actor.AnchorY
	if anchorX == 0 && anchorY == 0 {
		anchorX, anchorY = 55, 162
	}
	t.actor.X = x + float64(anchorX)*zoom/100.0
	t.actor.Y = y + float64(anchorY)*zoom/100.0
}

type loc04DoorWalkOutTask struct {
	ctx            *Context
	actor          *engine.Actor
	frameClock     float64
	transformClock float64
	transformN     int
}

func (t *loc04DoorWalkOutTask) Update(dt float64) bool {
	if t.actor == nil {
		actor, _, ok := t.ctx.session.actorByNameOrAsset(loc04Actor)
		if !ok || actor == nil {
			log.Printf(`game: scene "006": exit transform: required actor %q is missing`, loc04Actor)
			return true
		}
		t.actor = actor
		t.actor.Visible = true
		t.actor.Zoom = 72
		t.setFromTransform(223, 122, 72)
		t.actor.Frame = 0
	}

	t.transformClock += dt
	for t.transformClock >= 1.0/loc04EntranceTransformHz && t.transformN < 30 {
		t.transformClock -= 1.0 / loc04EntranceTransformHz
		t.transformN++
		p := float64(t.transformN) / 30.0
		x := 223.0 + (465.0-223.0)*p
		y := 122.0 + (100.0-122.0)*p
		zoom := 72.0 + (125.0-72.0)*p
		t.setFromTransform(x, y, zoom)
		t.actor.Zoom = int(zoom + 0.5)
	}

	t.frameClock += dt
	for t.frameClock >= 1.0/engine.DefaultWalkAnimFPS {
		t.frameClock -= 1.0 / engine.DefaultWalkAnimFPS
		if t.actor.Frame < 11 {
			t.actor.Frame++
		}
	}

	return t.transformN >= 30
}

func (t *loc04DoorWalkOutTask) setFromTransform(x, y, zoom float64) {
	anchorX, anchorY := t.actor.AnchorX, t.actor.AnchorY
	if anchorX == 0 && anchorY == 0 {
		anchorX, anchorY = 55, 162
	}
	t.actor.X = x + float64(anchorX)*zoom/100.0
	t.actor.Y = y + float64(anchorY)*zoom/100.0
}

func loc04FirstVisit(ctx *Context) engine.Task {
	return engine.Sequence(
		loc04SitNormal(ctx),
		loc04SpeakLayer(ctx, "RodTalk", "006_ROD_01", 0, 10, true),
		ctx.Wait(1),
		loc04SwitchBackground(ctx, 0),
		loc04PoseDesk(ctx),
	)
}

func loc04OfficeWalk(ctx *Context, actor string, x, y float64) engine.Task {
	return ctx.WalkToPerspective(actor, x, y)
}

func loc04OfficeWalkFacing(ctx *Context, actor string, x, y float64, direction int) engine.Task {
	return ctx.WalkToFacingPerspective(actor, x, y, direction)
}

func loc04SitNormal(ctx *Context) engine.Task {
	return engine.Sequence(loc04OfficeWalkFacing(ctx, loc04Actor, 0xAB, 0x103, 6), ctx.HideActor(loc04Actor), loc04PlayWhole(ctx, "RodSetzen"), loc04SwitchBackground(ctx, 1), loc04PoseRodTalk(ctx), ctx.SetFlag(loc04Seated, 1))
}

func loc04SitForCustomer(ctx *Context) engine.Task {
	return engine.Sequence(loc04OfficeWalkFacing(ctx, loc04Actor, 0xAB, 0xF7, 6), ctx.HideActor(loc04Actor), loc04PlayRange(ctx, "RodSetzenLayer", 0, 5), loc04SwitchBackground(ctx, 2), loc04HidePoseLayers(ctx), ctx.ShowLayer("RodSitAct1"), ctx.PlayLayerFrames("RodSitAct1", 0, -1), ctx.HideLayer("RodSitAct1"), ctx.SetFlag(loc04Seated, 1))
}

func loc04StandIfNeeded(ctx *Context) engine.Task {
	if ctx.GetFlag(loc04Seated) == 0 {
		return engine.Immediate(func() {})
	}
	return engine.Sequence(loc04SwitchBackground(ctx, 0), loc04HidePoseLayers(ctx), loc04PlayRange(ctx, "RodSetzenLayer", 0x14, 0), ctx.PlaceActor(loc04Actor, 0xAD, 0xF7), loc04PoseStanding(ctx), ctx.SetFlag(loc04Seated, 0))
}

func loc04CustomerArrival(ctx *Context) engine.Task {
	return engine.Sequence(
		loc04PlayRange(ctx, "RamilRein", 0, 5), ctx.PlaySFX("Sfx_Door_CreakShort.wav"),
		loc04PlayRange(ctx, "RamilRein", 6, 0x10), ctx.PlaySFX("Sfx_Door_CreakShort.wav"),
		loc04PlayRange(ctx, "RamilRein", 0x11, 0x17), ctx.PlaySFX("Sfx_Tock.wav"), loc04PlayRange(ctx, "RamilRein", 0x18, -1),
		loc04SwitchBackground(ctx, 0), loc04PoseDesk(ctx), ctx.PlaceActorPerspective(loc04CustomerActor, 0x240, 0x164), ctx.SetFlag(loc04CustomerPresent, 1), ctx.ShowActor(loc04CustomerActor),
		loc04SpeakRodMode(ctx, "006_ROD_02", 1), loc04OfficeWalkFacing(ctx, loc04CustomerActor, 0x82, 0x124, 5),
		loc04SpeakRam(ctx, "006_RAM_01"), loc04SetAreaMode(ctx, true), ctx.SetFlag(loc04CustomerStage, 1), ctx.SetFlag(FlagLoc04CustomerSeen, 1),
	)
}

func loc04SetAreaMode(ctx *Context, customer bool) engine.Task {
	return engine.Immediate(func() {
		for _, id := range loc04BaseAreas {
			if a, ok := ctx.session.scene.Areas[id]; ok {
				a.Enabled = !customer
			}
		}
		for _, id := range loc04CustomerAreas {
			if a, ok := ctx.session.scene.Areas[id]; ok {
				a.Enabled = customer
			}
		}
		if customer {
			order := ctx.session.scene.AreaOrder
			for i, id := range order {
				if id != "RamTalk" {
					continue
				}
				copy(order[1:i+1], order[:i])
				order[0] = "RamTalk"
				break
			}
		}
	})
}

func loc04HidePoseLayers(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		for _, id := range []string{"RodTalk", "RodDeskTalk", "RodSitAct1", "RodSitAct2", "RamTalk"} {
			if l, ok := ctx.session.scene.Layers[id]; ok {
				l.Visible = false
			}
		}
	})
}

func loc04SetIdleFrame(ctx *Context, ids ...string) engine.Task {
	return engine.Immediate(func() {
		for _, id := range ids {
			l, ok := loc04UsableLayer(ctx, id, "set idle frame")
			if !ok {
				continue
			}
			l.Visible = true
			l.Enabled = true
			l.Playing = false
			l.TaskDriven = false
			l.Frame = 0
			l.Accumulator = 0
		}
	})
}

func loc04PoseStanding(ctx *Context) engine.Task {
	return engine.Sequence(loc04HidePoseLayers(ctx), ctx.ShowActor(loc04Actor))
}
func loc04PoseDesk(ctx *Context) engine.Task {
	return loc04PoseOrActor(ctx, "RodDeskTalk")
}
func loc04PoseRodTalk(ctx *Context) engine.Task {
	return engine.Sequence(ctx.HideActor(loc04CustomerActor), loc04PoseOrActor(ctx, "RodTalk"))
}

func loc04PoseOrActor(ctx *Context, id string) engine.Task {
	return engine.Immediate(func() {
		for _, poseID := range []string{"RodTalk", "RodDeskTalk", "RamTalk", "RodSitAct1", "RodSitAct2"} {
			if l, ok := ctx.session.scene.Layers[poseID]; ok {
				l.Visible = false
			}
		}
		l, ok := loc04UsableLayer(ctx, id, "pose")
		a, _, actorOK := ctx.session.actorByNameOrAsset(loc04Actor)
		if !ok {
			if actorOK {
				a.Visible = true
			}
			return
		}
		if actorOK {
			a.Visible = false
		}
		l.Visible = true
		l.Enabled = true
		l.Playing = false
		l.TaskDriven = false
		l.Frame = 0
		l.Accumulator = 0
	})
}
func loc04PoseCustomer(ctx *Context) engine.Task {
	return engine.Sequence(ctx.HideActor(loc04Actor), ctx.HideActor(loc04CustomerActor), loc04HidePoseLayers(ctx), loc04SetIdleFrame(ctx, "RamTalk", "RodSitAct2"))
}
func loc04PoseCustomerAct1(ctx *Context) engine.Task {
	return engine.Sequence(ctx.HideActor(loc04Actor), ctx.HideActor(loc04CustomerActor), loc04HidePoseLayers(ctx), loc04SetIdleFrame(ctx, "RamTalk", "RodSitAct1"))
}

func loc04PlayRange(ctx *Context, id string, from, to int) engine.Task {
	if _, ok := loc04UsableLayer(ctx, id, "play range"); !ok {
		return engine.Immediate(func() {})
	}
	return engine.Sequence(ctx.ShowLayer(id), ctx.PlayLayerFrames(id, from, to), ctx.HideLayer(id))
}
func loc04PlayWhole(ctx *Context, id string) engine.Task { return loc04PlayRange(ctx, id, 0, -1) }
func loc04Speak(ctx *Context, actor, line string) engine.Task {
	return ctx.Say(actor, line, "["+line+"]")
}
func loc04SpeakAt(ctx *Context, x, y float64, direction int, line string) engine.Task {
	return engine.Sequence(loc04OfficeWalkFacing(ctx, loc04Actor, x, y, direction), loc04Speak(ctx, loc04Actor, line))
}

func loc04ContextSpeech(ctx *Context, x, y float64, direction int, line string) engine.Task {
	if ctx.GetFlag(loc04Seated) != 0 {
		return loc04SpeakRodMode(ctx, line, 1)
	}
	return loc04SpeakAt(ctx, x, y, direction, line)
}

func loc04SpeakLayer(ctx *Context, id, line string, start, end int, keep bool) engine.Task {
	layer, ok := loc04UsableLayer(ctx, id, "speech "+line)
	if !ok {
		return ctx.PlayVoiceover(line, "["+line+"]")
	}
	if start < 0 {
		start = 0
	}
	if end < 0 || end >= layer.Source.Frames() {
		end = layer.Source.Frames() - 1
	}
	tasks := []engine.Task{ctx.ShowLayer(id), ctx.PlaySpeechBoundToLayer(id, line, "["+line+"]", start, end), ctx.FreezeLayer(id, 0)}
	if !keep {
		tasks = append(tasks, ctx.HideLayer(id))
	}
	return engine.Sequence(tasks...)
}

func loc04SpeakRodMode(ctx *Context, line string, mode int) engine.Task {
	switch mode {
	case 1:
		return engine.Sequence(loc04SwitchBackground(ctx, 0), loc04PoseDesk(ctx), loc04SyncCustomerActor(ctx), loc04SpeakLayer(ctx, "RodDeskTalk", line, 0, 9, true))
	case 2:
		return engine.Sequence(loc04SwitchBackground(ctx, 2), loc04PoseCustomer(ctx), loc04SpeakLayer(ctx, "RodSitAct2", line, 0, -1, true))
	default:
		return engine.Sequence(loc04SwitchBackground(ctx, 1), loc04PoseRodTalk(ctx), loc04SpeakLayer(ctx, "RodTalk", line, 0, -1, true))
	}
}

func loc04SyncCustomerActor(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		if actor, _, ok := ctx.session.actorByNameOrAsset(loc04CustomerActor); ok {
			actor.Visible = ctx.GetFlag(loc04CustomerPresent) != 0
		}
	})
}

func loc04SpeakRam(ctx *Context, line string) engine.Task {
	return engine.Sequence(loc04SwitchBackground(ctx, 2), loc04PoseCustomer(ctx), loc04SpeakLayer(ctx, "RamTalk", line, 0, -1, true))
}

func loc04UsableLayer(ctx *Context, id, use string) (*engine.Layer, bool) {
	layer, ok := ctx.session.scene.Layers[id]
	if !ok || layer == nil || layer.Source == nil {
		autoLayer, real, err := ctx.ensureAssetLayer(id)
		if err == nil {
			layer = autoLayer
			ok = true
			log.Printf(`game: scene "006": %s: auto-detected layer %q from %q`, use, id, real)
		} else if !ok || layer == nil {
			log.Printf(`game: scene "006": %s: required layer %q is missing: %v`, use, id, err)
			return nil, false
		} else {
			log.Printf(`game: scene "006": %s: layer %q is broken: nil source; auto-detect failed: %v`, use, id, err)
			return nil, false
		}
	}
	if layer.Source == nil {
		log.Printf(`game: scene "006": %s: layer %q is broken: nil source`, use, id)
		return nil, false
	}
	frames := layer.Source.Frames()
	if frames <= 0 {
		log.Printf(`game: scene "006": %s: layer %q is broken: source reports %d frames`, use, id, frames)
		return nil, false
	}
	return layer, true
}

func loc04PrepareBackgrounds(ctx *Context, loadRodTalk, loadRamTalk bool) {
	runtime := &loc04Runtime{baseBackground: ctx.session.scene.Background}
	if loadRodTalk {
		if fragment, err := ctx.instantiateSceneFragment("006_rodtalk"); err != nil {
			log.Printf("game: scene fragment %q: %v", "006_rodtalk", err)
		} else {
			runtime.rodTalkBackground = fragment.Background
			loc04ImportLayer(ctx.session.scene, fragment, "RodTalk")
		}
	}
	if loadRamTalk {
		if fragment, err := ctx.instantiateSceneFragment("006_ramtalk"); err != nil {
			log.Printf("game: scene fragment %q: %v", "006_ramtalk", err)
		} else {
			runtime.ramTalkBackground = fragment.Background
			for _, id := range []string{"RamTalk", "RodSitAct1", "RodSitAct2"} {
				loc04ImportLayer(ctx.session.scene, fragment, id)
			}
			for _, id := range []string{"006_DoorRamil", "006_NewsRamil", "006_FassRamil"} {
				loc04ImportArea(ctx.session.scene, fragment, id)
			}
		}
	}
	ctx.session.loc04Runtime = runtime
	ctx.session.scene.Background = runtime.baseBackground
}

func loc04ImportLayer(dst, src *engine.Scene, id string) {
	if dst == nil || src == nil {
		return
	}
	layer, ok := src.Layers[id]
	if !ok || layer == nil {
		log.Printf("game: scene fragment: required layer %q is missing", id)
		return
	}
	if _, exists := dst.Layers[id]; !exists {
		dst.LayerOrder = append(dst.LayerOrder, id)
	}
	dst.Layers[id] = layer
}

func loc04ImportArea(dst, src *engine.Scene, id string) {
	if dst == nil || src == nil {
		return
	}
	area, ok := src.Areas[id]
	if !ok || area == nil {
		log.Printf("game: scene fragment: required area %q is missing", id)
		return
	}
	if _, exists := dst.Areas[id]; !exists {
		dst.AreaOrder = append(dst.AreaOrder, id)
	}
	dst.Areas[id] = area
}

func loc04SwitchBackground(ctx *Context, mode int) engine.Task {
	return engine.Immediate(func() {
		runtime := ctx.session.loc04Runtime
		if runtime == nil || ctx.session.scene == nil {
			return
		}
		background := runtime.baseBackground
		if mode == 1 && runtime.rodTalkBackground != nil {
			background = runtime.rodTalkBackground
		}
		if mode == 2 && runtime.ramTalkBackground != nil {
			background = runtime.ramTalkBackground
		}
		ctx.session.scene.Background = background
	})
}

func loc04AuditLayers(ctx *Context) {
	required := []string{"RodRein", "RodRaus", "RodSetzen", "RodSetzenLayer", "RodDeskTalk"}
	if ctx.GetFlag(FlagLoc04CustomerState) == 1 {
		required = append(required, "RodTalk")
		required = append(required, "RamTalk", "RodSitAct1", "RodSitAct2", "RamilRein", "RamilRaus", "RamilGeld")
	}
	for _, id := range required {
		loc04UsableLayer(ctx, id, "scene initialization")
	}
}

func loc04AdvanceCustomerConversation(ctx *Context) engine.Task {
	stage := ctx.GetFlag(loc04CustomerStage)
	var tasks []engine.Task
	next := stage + 1
	switch stage {
	case 2:
		tasks = []engine.Task{loc04SpeakRodMode(ctx, "006_ROD_16", 0), loc04SpeakRam(ctx, "006_RAM_06")}
	case 3:
		tasks = []engine.Task{loc04SpeakRodMode(ctx, "006_ROD_17", 0), loc04SpeakRam(ctx, "006_RAM_07"), loc04SpeakRodMode(ctx, "006_ROD_18", 2)}
	case 4:
		tasks = []engine.Task{loc04SpeakRodMode(ctx, "006_ROD_19", 0), loc04SpeakRam(ctx, "006_RAM_08")}
	case 5:
		tasks = []engine.Task{loc04SpeakRodMode(ctx, "006_ROD_20", 0), loc04SpeakRam(ctx, "006_RAM_09")}
	case 6:
		tasks = []engine.Task{loc04SpeakRodMode(ctx, "006_ROD_21", 1), ctx.PlayVoiceover("006_RAM_10", "[006_RAM_10]"), ctx.ShowLayer("RamilGeld"), ctx.PlayLayerFrames("RamilGeld", 0, 0x19), ctx.PlaySFX("Sfx_TockSoft.wav"), ctx.PlayLayerFrames("RamilGeld", 0x1A, 0x2E), ctx.PlaySFX("Sfx_MoneyBag.wav"), ctx.PlayLayerFrames("RamilGeld", 0x2F, 0x3B), ctx.PlaySFX("Sfx_Tock.wav"), ctx.PlayLayerFrames("RamilGeld", 0x3C, -1), ctx.HideLayer("RamilGeld"), ctx.SetFlag(FlagLoc04PaymentOnDesk, 1), loc04SwitchBackground(ctx, 2), loc04PoseCustomer(ctx)}
	case 7:
		tasks = []engine.Task{loc04SpeakRodMode(ctx, "006_ROD_22", 0), loc04SpeakRam(ctx, "006_RAM_11")}
	case 8:
		tasks = []engine.Task{loc04SpeakRodMode(ctx, "006_ROD_23", 2), loc04SpeakRam(ctx, "006_RAM_12")}
	case 9:
		next = 0
		tasks = []engine.Task{loc04SpeakRodMode(ctx, "006_ROD_24", 1), loc04SetAreaMode(ctx, false), ctx.PlaceActorPerspective(loc04CustomerActor, 0x82, 0x124), loc04SyncCustomerActor(ctx), loc04OfficeWalkFacing(ctx, loc04CustomerActor, 0x240, 0x164, 6), ctx.HideActor(loc04CustomerActor), ctx.SetFlag(loc04CustomerPresent, 0), ctx.ShowLayer("RamilRaus"), ctx.PlayLayerFrames("RamilRaus", 0, 5), ctx.PlaySFX("Sfx_Door_CreakShort.wav"), ctx.PlayLayerFrames("RamilRaus", 6, -1), ctx.PlaySFX("Sfx_Tock.wav"), ctx.HideLayer("RamilRaus"), loc04PoseDesk(ctx), ctx.SetFlag(FlagLoc04CustomerState, 2), loc04SpeakRodMode(ctx, "006_ROD_25", 1)}
	default:
		next = 2
		tasks = []engine.Task{loc04SpeakRodMode(ctx, "006_ROD_13", 0), loc04SpeakRam(ctx, "006_RAM_03"), loc04SpeakRodMode(ctx, "006_ROD_14", 2), loc04SpeakRam(ctx, "006_RAM_04"), ctx.SetFlag(FlagLoc04CustomerStory, 1), loc04SpeakRodMode(ctx, "006_ROD_15", 2), loc04SpeakRam(ctx, "006_RAM_05")}
	}
	tasks = append(tasks, ctx.SetFlag(loc04CustomerStage, next))
	return engine.Sequence(tasks...)
}
