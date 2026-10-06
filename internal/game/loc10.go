package game

import (
	"encoding/binary"
	"math"
	"math/rand/v2"
	"strings"

	"github.com/wok/rent-a-hero/internal/audio"
	"github.com/wok/rent-a-hero/internal/engine"
)

const (
	loc10StateNight     = 0x3e80
	loc10StateGlider    = 0x3f24
	loc10StateJas       = 0x3f28
	loc10StateWald      = 0x3f2c
	loc10StateGirl      = 0x3f30
	loc10StateGirlTalk  = 0x3f34
	loc10StateTo57      = 0x3f38
	loc10StateStory     = 0x3f3c
	loc10StateTo99      = 0x3f40
	loc10StateMushroom  = 0x3f44
	loc10StateTree      = 0x3f48
	loc10StateTreeTalk  = 0x3f4c
	loc10StateTreeStage = 0x3f50
	loc10StateStraw     = 0x3f54
	loc10StateNecklace  = 0x3f58
	loc10StateTeddy     = 0x3f5c
	loc10StateTo91      = 0x3f60
	loc10StateSlug      = 0x3f64
	loc10StateLeaf      = 0x3f68
	loc10StateLeafReady = 0x3f6c
)

type LOC10Controller struct{}

func (LOC10Controller) Enter(ctx *Context, scene, from string) engine.Task {
	scene = loc10SceneID(scene)
	actor := loc10ActorID(ctx)
	if scene == "S40" {
		loc10PrepareScene(ctx, scene)
	}
	tasks := []engine.Task{engine.Immediate(func() {
		if scene != "S40" {
			loc10PrepareScene(ctx, scene)
		}
		loc10SyncJasminActor(ctx)
	})}
	music := "Loc10_ForestBeforeDarkness.wav"
	if loc10Original(ctx, loc10StateNight) != 0 {
		music = "Loc10_ForestInDarkness.wav"
	}
	if scene == "S66" {
		music = "Loc35_RodrigoOnGlider.wav"
	}
	if loc10Original(ctx, loc10StateNight) != 0 && scene != "S66" {
		tasks = append(tasks, ctx.SetBackgroundByStem(scene+"_Back_nd"))
	}
	tasks = append(tasks, ctx.PlayMusic(music))

	var entry engine.Task
	switch scene {
	case "S37":
		entry = loc10Enter37(ctx, actor, from)
	case "S38":
		entry = loc10Enter38(ctx, actor, from)
	case "S39":
		entry = loc10Enter39(ctx, actor, from)
	case "S40":
		entry = loc10Enter40(ctx, actor, from)
	case "S41":
		entry = loc10Enter41(ctx, actor, from)
	case "S42":
		entry = loc10Enter42(ctx, actor, from)
	case "S43":
		entry = loc10Enter43(ctx, actor, from)
	case "S44":
		entry = loc10Enter44(ctx, actor, from)
	case "S66":
		if layer, _, err := ctx.ensureAssetLayer("S66_WoodChase"); err == nil {
			layer.Presentation = true
			layer.ColorKeyed = false
		}
		entry = loc10WoodChase(ctx)
	}
	if entry != nil {
		tasks = append(tasks, entry)
	}
	if scene != "S66" {
		tasks = append(tasks, ctx.RunAmbient(&loc10Ambient{ctx: ctx}))
	}
	return engine.Sequence(tasks...)
}

func (LOC10Controller) Exit(*Context, string, string) engine.Task { return nil }

func (LOC10Controller) Click(ctx *Context, area string) engine.Task {
	actor := loc10ActorID(ctx)
	switch area {
	case "S37_To38":
		return loc10Nav37(ctx, actor, 1)
	case "S37_To39":
		return loc10Nav37(ctx, actor, 2)
	case "S37_To40":
		return loc10Nav37(ctx, actor, 3)
	case "S37_To41":
		return loc10Nav37(ctx, actor, 4)
	case "S37_To42":
		return loc10Nav37(ctx, actor, 5)
	case "S37_To43":
		return loc10Nav37(ctx, actor, 6)
	case "S37_Pilz":
		return loc10WalkVoice(ctx, actor, 0xdd, 0x138, 1, "037_ROD_03")
	case "S37_Wald":
		line := "037_ROD_01"
		if loc10Original(ctx, loc10StateWald) != 0 {
			line = "037_ROD_02"
		}
		return loc10WalkVoice(ctx, actor, 0xaa, 199, 4, line)
	case "S37_Glider":
		return loc10Leave37ByGlider(ctx, actor)

	case "S38_To37":
		return loc10Nav38(ctx, actor, 0x14)
	case "S38_To39":
		return loc10Nav38(ctx, actor, 0x15)
	case "S38_To57":
		if loc10Original(ctx, loc10StateTo57) != 0 {
			return loc10Nav38(ctx, actor, 0x16)
		}
	case "S38_To68":
		if loc10Original(ctx, loc10StateStory) != 0 {
			return loc10Nav38(ctx, actor, 0x17)
		}
	case "S38_Pilz":
		return loc10WalkVoice(ctx, actor, 0x1c3, 0xc1, 5, "038_ROD_08")
	case "S38_Wald":
		return loc10WalkVoice(ctx, actor, 0x21b, 0xbf, 4, "038_ROD_01")
	case "S38_Girl", "S38_GirlJumps":
		if loc10Original(ctx, loc10StateGirl) != 0 {
			return loc10GirlTalk(ctx, actor)
		}

	case "S39_To37":
		return loc10WalkScene(ctx, actor, 7, 0x124, 1, "S37")
	case "S39_To38":
		return loc10WalkScene(ctx, actor, 5, 0xd3, 1, "S38")
	case "S39_To40":
		return loc10WalkScene(ctx, actor, 0x16d, 0x163, 0, "S40")
	case "S39_To99":
		if loc10Original(ctx, loc10StateTo99) != 0 {
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x27b, 0xbe, 6), ctx.ChangeLocation(8, "99"))
		}
	case "S39_Pilz1":
		return loc10WalkVoice(ctx, actor, 0x1c7, 0xe4, 7, "039_ROD_01")
	case "S39_Pilz2":
		return loc10WalkVoice(ctx, actor, 0x1c7, 0xe4, 7, "039_ROD_02")
	case "S39_Pilz3":
		return loc10WalkVoice(ctx, actor, 0x1c7, 0xe4, 7, "039_ROD_03")
	case "S39_Wald":
		return loc10WalkVoice(ctx, actor, 0x165, 0xa6, 4, "039_ROD_FX1")
	case "S39_Pilz":
		return loc10TakeMushroom(ctx, actor)

	case "S40_To37":
		return loc10WalkScene(ctx, actor, 5, 0xcd, 2, "S37")
	case "S40_To39":
		return loc10WalkScene(ctx, actor, 0x13a, 0x4c, 4, "S39")
	case "S40_Pilze":
		return loc10WalkVoice(ctx, actor, 0xb0, 0xe7, 1, "040_ROD_11")
	case "S40_Pilz":
		return loc10WalkVoice(ctx, actor, 0x17d, 0xf7, 0, "040_ROD_12")
	case "S40_Wald":
		return loc10WalkVoice(ctx, actor, 0xda, 0xb0, 3, "040_ROD_01")
	case "S40_Baum":
		return loc10TreeTalk(ctx, actor)
	case "S40_Root":
		return loc10RootLook(ctx, actor)
	case "S40_Necklace":
		return loc10TakeNecklace(ctx, actor)
	case "S40_Straw":
		return loc10TakeStraw(ctx, actor)
	case "S40_FireLoops":
		if loc10Original(ctx, loc10StateStraw) == 3 {
			return loc10KillFire(ctx, actor)
		}

	case "S41_To37":
		return loc10WalkScene(ctx, actor, 0x15a, 0x60, 4, "S37")
	case "S41_To42":
		return loc10WalkScene(ctx, actor, 5, 200, 2, "S42")
	case "S41_Pilz":
		return loc10WalkVoice(ctx, actor, 0x3c, 0xcb, 4, "040_ROD_11")
	case "S41_Teddy", "S41_Wald":
		return loc10Teddy(ctx, actor)

	case "S42_To37":
		return loc10WalkScene(ctx, actor, 0x274, 0x77, 5, "S37")
	case "S42_To41":
		return loc10WalkScene(ctx, actor, 0x274, 0xcb, 6, "S41")
	case "S42_To91":
		if loc10Original(ctx, loc10StateTo91) != 0 {
			return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x135, 0x163, 0), ctx.ChangeLocation(12, "91"))
		}
	case "S42_Wald":
		return loc10WalkVoice(ctx, actor, 0xf7, 0xad, 4, "042_ROD_01")

	case "S43_To37":
		return loc10WalkScene(ctx, actor, 0x27b, 0xc0, 6, "S37")
	case "S43_To44":
		return loc10WalkScene(ctx, actor, 0x12f, 0x78, 4, "S44")
	case "S43_Wald":
		return loc10WalkVoice(ctx, actor, 0xcf, 0xb4, 3, "043_ROD_01")
	case "S43_Pilz":
		return loc10WalkVoice(ctx, actor, 0x118, 0xf7, 0, "043_ROD_02")

	case "S44_To43":
		return loc10WalkScene(ctx, actor, 0x148, 0x163, 0, "S43")
	case "S44_Wald":
		return loc10WalkVoice(ctx, actor, 0xa8, 0xb8, 4, "044_ROD_01")
	case "S44_Busch":
		return loc10WalkVoice(ctx, actor, 0x125, 0xb8, 4, "044_ROD_02")
	case "S44_Pilz":
		return loc10WalkVoice(ctx, actor, 0x1b1, 0xb9, 6, "044_ROD_04")
	case "S44_Leaf":
		return loc10TakeLeaf(ctx, actor)
	}
	return nil
}

func (LOC10Controller) UseItem(*Context, int, string) engine.Task { return nil }
func (LOC10Controller) SelectItem(ctx *Context, item int) engine.Task {
	actor := loc10ActorID(ctx)
	switch ctx.session.state.Scene {
	case "S38":
		if item == 12 && loc10Original(ctx, loc10StateGirl) != 0 {
			return loc10OfferTeddy(ctx, actor)
		}
	case "S40":
		if item == 15 && loc10Original(ctx, loc10StateTree) != 0 && loc10Original(ctx, loc10StateStraw) == 1 {
			return loc10DropStraw(ctx, actor)
		}
		if item == 26 && loc10Original(ctx, loc10StateTree) != 0 && loc10Original(ctx, loc10StateStraw) == 2 {
			return loc10SetFire(ctx, actor)
		}
	case "S44":
		if item == 8 {
			return loc10Slug(ctx, actor)
		}
	}
	return nil
}

func (LOC10Controller) LoadConditionMask(ctx *Context, scene string) int {
	scene = loc10SceneID(scene)
	switch scene {
	case "S37":
		m := 0
		from := loc10SceneID(ctx.session.state.PreviousScene)
		if from == "S66" {
			m |= 1
		}
		if from == "S113" || ctx.session.state.PreviousLocation == 31 {
			m |= 2 | 4
			loc10SetOriginal(ctx, loc10StateGlider, 1)
		} else if loc10Original(ctx, loc10StateGlider) != 0 {
			m |= 4
		}
		return m
	case "S38":
		m := 0
		if loc10Original(ctx, loc10StateJas) != 0 {
			m |= 1
		}
		if loc10Original(ctx, loc10StateGirl) != 0 {
			m |= 2
			if ctx.HasItem(12) {
				m |= 4
			}
		}
		return m
	case "S40":
		if loc10Original(ctx, loc10StateTree) != 0 {
			return 1
		}
	case "S44":
		if loc10Original(ctx, loc10StateLeaf) != 0 {
			return 1
		}
	}
	return 0
}

func loc10Enter37(ctx *Context, actor, from string) engine.Task {
	from = loc10SceneID(from)
	if from == "S113" || ctx.session.state.PreviousLocation == 31 {
		loc10SetOriginal(ctx, loc10StateGlider, 1)
		return engine.Sequence(ctx.HideActor(actor), ctx.HideLayer("S37_Glider"), ctx.PlaySFXVolume("Sfx_Glider_PassingBy.wav", 0x19), loc10PlayLayerFramesDeferred(ctx, "S37_GliderArrives", 0, -1), ctx.HideLayer("S37_GliderArrives"), ctx.PlaceActorPerspective(actor, 0xfa, 0xc5), ctx.ShowActor(actor), ctx.WalkToFacingPerspective(actor, 0xfa, 200, 0), loc10ShowGlider(ctx))
	}
	if from == "S66" {
		loc10SetOriginal(ctx, loc10StateWald, 1)
		loc10SetOriginal(ctx, loc10StateJas, 1)
		loc10SetOriginal(ctx, loc10StateStory, 1)
		loc10SetOriginal(ctx, 0x3f04, 1)
		loc10SetOriginal(ctx, 0x3efc, 0)
		loc10SetOriginal(ctx, 0x3f00, 0)
		jas := loc10JasminActorID(ctx)
		return engine.Sequence(
			ctx.HideActor(actor),
			ctx.HideActor(jas),
			ctx.ShowLayer("S37_JasFallDown"),
			ctx.RunAmbient(ctx.PlayLayer("S37_JasFallDown")),
			loc10PlayLayerFramesDeferred(ctx, "S37_RodFallDown", 0, 7),
			ctx.PlaySFX("Sfx_Knock_Wet.wav"),
			loc10PlayLayerFramesDeferred(ctx, "S37_RodFallDown", 8, -1),
			ctx.HideLayer("S37_RodFallDown"),
			ctx.PlaceActorPerspective(actor, 0xa3, 0xcc),
			ctx.ShowActor(actor),
			ctx.WalkToFacingPerspective(actor, 0xdc, 0xcc, 6),
			ctx.PlayVoiceover("037_JAS_01", "[037_JAS_01]"),
			ctx.HideLayer("S37_JasFallDown"),
			ctx.PlaceActorPerspective(jas, 0xfe, 0xbb),
			ctx.ShowActor(jas),
			ctx.WalkToFacingPerspective(jas, 0x149, 0xe2, 2),
			ctx.SetActorOrientation(jas, 4),
			ctx.HideActor(jas),
			ctx.ShowLayer("S37_JasPointN"),
		)
	}
	var t engine.Task
	switch from {
	case "S38":
		t = engine.Sequence(ctx.PlaceActorPerspective(actor, 0x145, 0x70), ctx.WalkToFacingPerspective(actor, 0x145, 0x82, 0))
	case "S39":
		t = engine.Sequence(ctx.PlaceActorPerspective(actor, 0x27a, 0x87), ctx.WalkToFacingPerspective(actor, 0x23f, 0x96, 1))
	case "S40":
		t = engine.Sequence(ctx.PlaceActorPerspective(actor, 0x276, 0xcb), ctx.WalkToFacingPerspective(actor, 0x24f, 0xc6, 2))
	case "S41":
		t = engine.Sequence(ctx.PlaceActorPerspective(actor, 0x135, 0x161), ctx.WalkToFacingPerspective(actor, 0x140, 0x145, 4))
	case "S42":
		t = engine.Sequence(ctx.PlaceActorPerspective(actor, 10, 0x101), ctx.WalkToFacingPerspective(actor, 0x59, 0xff, 6))
	case "S43":
		t = engine.Sequence(ctx.PlaceActorPerspective(actor, 10, 0xc3), ctx.WalkToFacingPerspective(actor, 0x4f, 199, 6))
	}
	if loc10Original(ctx, loc10StateGlider) != 0 {
		if t != nil {
			return engine.Sequence(t, loc10ShowGlider(ctx))
		}
		return loc10ShowGlider(ctx)
	}
	return t
}

func loc10ShowGlider(ctx *Context) engine.Task {
	return engine.Sequence(ctx.ShowLayer("S37_Glider"), ctx.MakeLayerClickable("S37_Glider"), ctx.RunAmbient(&loc10GliderIdle{ctx: ctx}))
}
func loc10Enter38(ctx *Context, actor, from string) engine.Task {
	from = loc10SceneID(from)
	tasks := make([]engine.Task, 0, 4)
	if loc10Original(ctx, loc10StateGirl) != 0 {
		tasks = append(tasks, ctx.MakeLayerClickable("S38_GirlJumps"), ctx.RunAmbient(&loc10GirlJumpAmbient{ctx: ctx, wait: 1 + rand.Float64()}))
	}
	var entry engine.Task
	switch from {
	case "S37":
		if loc10Original(ctx, loc10StateJas) != 0 {
			jas := loc10JasminActorID(ctx)
			entry = engine.Sequence(
				ctx.PlaceActorPerspective(jas, 0x149, 0x159),
				ctx.RunAmbient(ctx.WalkToFacingPerspective(jas, 0xc5, 0xc6, 2)),
				ctx.PlaceActorPerspective(actor, 0x151, 0x14a),
				ctx.WalkToFacingPerspective(actor, 0x173, 0xd2, 2),
			)
		} else {
			entry = engine.Sequence(ctx.PlaceActorPerspective(actor, 0x151, 0x163), ctx.WalkToFacingPerspective(actor, 0x157, 0x148, 4))
		}
	case "S39":
		entry = engine.Sequence(ctx.PlaceActorPerspective(actor, 0x27b, 0xc3), ctx.WalkToFacingPerspective(actor, 0x240, 0xc4, 2))
	case "S57":
		entry = engine.Sequence(ctx.PlaceActorPerspective(actor, 0x13e, 0x6c), ctx.WalkToFacingPerspective(actor, 0x13e, 0x7c, 0))
	case "S68", "S101":
		entry = engine.Sequence(ctx.PlaceActorPerspective(actor, 10, 0x88), ctx.WalkToFacingPerspective(actor, 0x67, 0xa5, 7))
	}
	if entry != nil {
		tasks = append(tasks, entry)
	}
	if len(tasks) == 0 {
		return nil
	}
	return engine.Sequence(tasks...)
}
func loc10Enter39(ctx *Context, actor, from string) engine.Task {
	from = loc10SceneID(from)
	tasks := make([]engine.Task, 0, 2)
	if loc10Original(ctx, loc10StateMushroom) != 0 {
		tasks = append(tasks, ctx.MakeLayerClickable("S39_Pilz"))
	}
	var entry engine.Task
	switch from {
	case "S37":
		entry = engine.Sequence(ctx.PlaceActorPerspective(actor, 7, 0x124), ctx.WalkToFacingPerspective(actor, 0x37, 0x123, 5))
	case "S38":
		entry = engine.Sequence(ctx.PlaceActorPerspective(actor, 5, 0xd3), ctx.WalkToFacingPerspective(actor, 0x35, 0xd8, 6))
	case "S40":
		entry = engine.Sequence(ctx.PlaceActorPerspective(actor, 0x16d, 0x163), ctx.WalkToFacingPerspective(actor, 0x176, 0x14f, 4))
	case "S99":
		entry = engine.Sequence(ctx.PlaceActorPerspective(actor, 0x27b, 0xbe), ctx.WalkToFacingPerspective(actor, 0x23c, 0xc2, 2))
	}
	if entry != nil {
		tasks = append(tasks, entry)
	}
	if len(tasks) == 0 {
		return nil
	}
	return engine.Sequence(tasks...)
}
func loc10Enter40(ctx *Context, actor, from string) engine.Task {
	from = loc10SceneID(from)
	tasks := []engine.Task{
		ctx.DisableArea("S40_Root"),
	}
	if from == "S37" {
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 5, 0xcd), ctx.WalkToFacingPerspective(actor, 0x37, 0xd3, 7))
	} else if from == "S39" {
		tasks = append(tasks, ctx.PlaceActorPerspective(actor, 0x13a, 0x4c), ctx.WalkToFacingPerspective(actor, 0x13e, 0x80, 0))
	}
	if loc10Original(ctx, loc10StateTree) != 0 {
		if loc10Original(ctx, loc10StateTreeStage) < 2 {
			tasks = append(tasks, ctx.RunAmbient(&loc10S40BranchTask{ctx: ctx, remaining: 20}))
		}
		switch loc10Original(ctx, loc10StateStraw) {
		case 1:
			tasks = append(tasks, ctx.EnableArea("S40_Root"))
		case 2:
			tasks = append(tasks, ctx.ShowLayer("S40_Straw"), ctx.MakeLayerClickable("S40_Straw"))
		case 3:
			tasks = append(tasks, loc10LoopLayer(ctx, "S40_FireLoops"), ctx.RunAmbient(&loc10S40FireSoundTask{ctx: ctx}), ctx.MakeLayerClickable("S40_FireLoops"))
		case 4:
			tasks = append(tasks, ctx.ShowLayer("S40_Straw"))
		}
		if loc10Original(ctx, loc10StateNecklace) != 0 {
			tasks = append(tasks, ctx.ShowLayer("S40_Necklace"), ctx.MakeLayerClickable("S40_Necklace"))
		}
	}
	return engine.Sequence(tasks...)
}

func loc10Enter41(ctx *Context, actor, from string) engine.Task {
	from = loc10SceneID(from)
	if from == "S42" {
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 5, 200), ctx.WalkToFacingPerspective(actor, 0x36, 0xcb, 7))
	}
	return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x15a, 0x60), ctx.WalkToFacingPerspective(actor, 0x151, 0x88, 0))
}
func loc10Enter42(ctx *Context, actor, from string) engine.Task {
	from = loc10SceneID(from)
	switch from {
	case "S37":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x274, 0x77), ctx.WalkToFacingPerspective(actor, 0x23c, 0x8f, 1))
	case "S41":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x274, 0xcb), ctx.WalkToFacingPerspective(actor, 0x252, 0xcb, 2))
	case "S91":
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x135, 0x163), ctx.WalkToFacingPerspective(actor, 0x143, 0x14d, 4))
	}
	return nil
}
func loc10Enter43(ctx *Context, actor, from string) engine.Task {
	from = loc10SceneID(from)
	if from == "S44" {
		return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x12f, 0x78), ctx.WalkToFacingPerspective(actor, 0x138, 0x8a, 0))
	}
	return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x27b, 0xc0), ctx.WalkToFacingPerspective(actor, 0x24c, 0xc4, 2))
}
func loc10Enter44(ctx *Context, actor, from string) engine.Task {
	return engine.Sequence(ctx.PlaceActorPerspective(actor, 0x176, 0x163), ctx.WalkToFacingPerspective(actor, 0x176, 0x13e, 4), engine.Immediate(func() {
		if loc10Original(ctx, loc10StateLeaf) != 0 && loc10Original(ctx, loc10StateLeafReady) != 0 {
			if l, ok := ctx.layer("S44_Leaf"); ok {
				l.Visible = true
			}
		}
	}))
}

func loc10Nav37(ctx *Context, actor string, n int) engine.Task {
	if loc10Original(ctx, loc10StateJas) == 0 {
		p := map[int][4]float64{1: {0x145, 0x70, 4, 38}, 2: {0x27a, 0x87, 5, 39}, 3: {0x276, 0xcb, 6, 40}, 4: {0x135, 0x161, 0, 41}, 5: {10, 0x101, 1, 42}, 6: {10, 0xc3, 2, 43}}[n]
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, p[0], p[1], int(p[2])), ctx.ChangeScene("S"+itoa10(int(p[3]))))
	}
	p := map[int][6]float64{
		1: {0x14e, 0x93, 4, 0x145, 0x70, 4},
		2: {0x226, 0x9c, 5, 0x184, 0xc6, 1},
		3: {0x236, 0xce, 6, 0x1a5, 0xd3, 2},
		4: {0x139, 0x12a, 0, 0x10f, 0xe9, 5},
		5: {0xc2, 0xeb, 1, 0xf0, 0xd5, 6},
		6: {0xbb, 0xca, 2, 0xe4, 0xcd, 6},
	}[n]
	if n == 1 {
		jas := loc10JasminActorID(ctx)
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, p[0], p[1], int(p[2])),
			ctx.HideLayer("S37_JasPointN"),
			ctx.RunAmbient(ctx.WalkToFacingPerspective(jas, 0x145, 0x70, 4)),
			ctx.WalkToFacingPerspective(actor, 0x145, 0x70, 4),
			ctx.ChangeScene("S38"),
		)
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, p[0], p[1], int(p[2])),
		ctx.PlayVoiceover("037_JAS_02", "[037_JAS_02]"),
		loc10PlayLayerDeferred(ctx, "S37_JasPointN"),
		ctx.WalkToFacingPerspective(actor, p[3], p[4], int(p[5])),
	)
}

func loc10Nav38(ctx *Context, actor string, e int) engine.Task {
	if loc10Original(ctx, loc10StateJas) != 0 {
		jas := loc10JasminActorID(ctx)
		if e == 0x17 {
			return engine.Sequence(
				ctx.WalkToFacingPerspective(actor, 0x6e, 0xa8, 3),
				ctx.HideLayer("S38_JasPointsNW"),
				ctx.RunAmbient(ctx.WalkToFacingPerspective(jas, 10, 0x88, 3)),
				ctx.WalkToFacingPerspective(actor, 10, 0x88, 3),
				ctx.HideActor(jas),
				ctx.ChangeLocation(19, "68"),
			)
		}
		p := map[int][6]float64{
			0x14: {0x14a, 299, 0, 0x14d, 0xf8, 3},
			0x15: {0x213, 199, 6, 0x186, 0xc3, 2},
			0x16: {0x142, 0x99, 4, 0x13b, 0xae, 1},
		}[e]
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, p[0], p[1], int(p[2])),
			ctx.PlayVoiceover("038_JAS_01", "[038_JAS_01]"),
			ctx.HideActor(jas),
			loc10PlayLayerDeferred(ctx, "S38_JasPointsNW"),
			ctx.WalkToFacingPerspective(actor, p[3], p[4], int(p[5])),
		)
	}
	switch e {
	case 0x14:
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x151, 0x163, 0), ctx.ChangeScene("S37"))
	case 0x15:
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x27b, 0xc3, 6), ctx.ChangeScene("S39"))
	case 0x16:
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x13e, 0x6c, 4), ctx.ChangeLocation(15, "57"))
	case 0x17:
		dest := "68"
		if loc10Original(ctx, loc10StateNight) != 0 {
			dest = "101"
		}
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 10, 0x88, 3), ctx.ChangeLocation(19, dest))
	}
	return nil
}

func loc10Leave37ByGlider(ctx *Context, actor string) engine.Task {
	if loc10Original(ctx, loc10StateGlider) == 0 {
		return nil
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0xfa, 0xc5, 4), ctx.HideLayer("S37_Glider"), ctx.HideActor(actor), loc10PlayLayerFramesDeferred(ctx, "S37_GliderLeaves", 0, 0x2a), ctx.PlaySFXVolume("Sfx_Glider_PassingBy.wav", 0x19), loc10PlayLayerFramesDeferred(ctx, "S37_GliderLeaves", 0x2b, -1), engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateGlider, 0) }), ctx.ChangeLocation(1, "8"))
}
func loc10GirlTalk(ctx *Context, actor string) engine.Task {
	s := loc10Original(ctx, loc10StateGirlTalk)
	if s < 1 {
		s = 1
	}
	base := []engine.Task{ctx.WalkToFacingPerspective(actor, 0x152, 0xda, 2)}
	switch s {
	case 1:
		return engine.Sequence(append(base,
			ctx.Say(actor, "038_ROD_02", "[038_ROD_02]"),
			loc10GirlSpeak(ctx, "038_MAED_02"),
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateGirlTalk, 2) }),
		)...)
	case 2:
		return engine.Sequence(append(base,
			ctx.Say(actor, "038_ROD_03", "[038_ROD_03]"),
			loc10GirlSpeak(ctx, "038_MAED_03"),
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateGirlTalk, 3) }),
		)...)
	case 3:
		return engine.Sequence(append(base,
			ctx.Say(actor, "038_ROD_04", "[038_ROD_04]"),
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateGirlTalk, 4) }),
		)...)
	case 4:
		return engine.Sequence(append(base,
			ctx.Say(actor, "038_ROD_05", "[038_ROD_05]"),
			loc10GirlSpeak(ctx, "038_MAED_05"),
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateGirlTalk, 5) }),
		)...)
	case 5:
		return engine.Sequence(append(base,
			ctx.Say(actor, "038_ROD_06", "[038_ROD_06]"),
			loc10GirlSpeak(ctx, "038_MAED_06"),
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateGirlTalk, 6) }),
		)...)
	case 6:
		return engine.Sequence(append(base,
			ctx.HideLayer("S38_GirlJumps"),
			loc10PlayLayerFramesDeferred(ctx, "S38_RodStealsRope", 0, 0x2b),
			ctx.PlaySFX("Sfx_Knock_Wet.wav"),
			ctx.PlayVoiceover("038_ROD_07", "[038_ROD_07]"),
			loc10PlayLayerFramesDeferred(ctx, "S38_RodStealsRope", 0x2c, -1),
			ctx.ShowLayer("S38_GirlJumps"),
			ctx.Say(actor, "038_ROD_11", "[038_ROD_11]"),
		)...)
	}
	return nil
}

func loc10GirlSpeak(ctx *Context, line string) engine.Task {
	return engine.Sequence(
		engine.Immediate(func() {
			if l, ok := ctx.layer("S38_GirlJumps"); ok {
				l.Playing = false
				l.TaskDriven = false
				l.Visible = false
				l.Enabled = false
			}
			if l, ok := ctx.layer("S38_Girl"); ok {
				l.Frame = 0
				l.Accumulator = 0
				l.Playing = false
				l.TaskDriven = false
				l.Visible = true
				l.Enabled = true
			}
		}),
		ctx.PlaySpeechBoundToLayer("S38_Girl", line, "["+line+"]", 0, 8),
		engine.Immediate(func() {
			if l, ok := ctx.layer("S38_Girl"); ok {
				l.Playing = false
				l.TaskDriven = false
				l.Visible = false
				l.Enabled = false
			}
			if l, ok := ctx.layer("S38_GirlJumps"); ok {
				l.Frame = 0
				l.Accumulator = 0
				l.Playing = false
				l.TaskDriven = false
				l.Visible = true
				l.Enabled = true
			}
		}),
	)
}

type loc10GirlJumpAmbient struct {
	ctx    *Context
	wait   float64
	active engine.Task
}

func (t *loc10GirlJumpAmbient) Update(dt float64) bool {
	if t.ctx.session.state.Location != 10 || loc10SceneID(t.ctx.session.state.Scene) != "S38" || loc10Original(t.ctx, loc10StateGirl) == 0 {
		return true
	}
	if t.active != nil {
		if l, ok := t.ctx.layer("S38_GirlJumps"); !ok || !l.Visible || !l.Enabled {
			if ok {
				l.Playing = false
				l.TaskDriven = false
			}
			t.active = nil
			t.wait = 5 + rand.Float64()*5
			return false
		}
		if t.active.Update(dt) {
			t.active = nil
			t.wait = 5 + rand.Float64()*5
		}
		return false
	}
	if l, ok := t.ctx.layer("S38_GirlJumps"); !ok || !l.Visible || !l.Enabled || l.TaskDriven {
		return false
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	t.active = loc10PlayLayerDeferred(t.ctx, "S38_GirlJumps")
	return false
}

func loc10OfferTeddy(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x152, 0xda, 2),
		ctx.Say(actor, "038_ROD_09", "[038_ROD_09]"),
		ctx.HideActor(actor),
		loc10HideAllActors(ctx),
		engine.Immediate(func() {
			for _, id := range []string{"S38_Girl", "S38_GirlJumps"} {
				if l, ok := ctx.layer(id); ok {
					l.Playing = false
					l.TaskDriven = false
					l.Visible = false
					l.Enabled = false
				}
			}
		}),
		loc10PlayLayerFramesDeferred(ctx, "S38_RodOffersTeddy", 0, 0x11),
		loc10PlayLayerFramesDeferred(ctx, "S38_RodOffersTeddy", 0x12, 0x17),
		ctx.PlayVoiceover("038_MAED_09", "[038_MAED_09]"),
		loc10PlayLayerFramesDeferred(ctx, "S38_RodOffersTeddy", 0x18, 0x23),
		ctx.PlayVoiceover("038_ROD_10", "[038_ROD_10]"),
		ctx.PlayVoiceover("038_MAED_10", "[038_MAED_10]"),
		loc10PlayLayerFramesDeferred(ctx, "S38_RodOffersTeddy", 0x24, 0x37),
		ctx.PlaySFX("Sfx_Step_Soft.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S38_RodOffersTeddy", 0x38, 0x3d),
		ctx.PlaySFX("Sfx_Step_Soft.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S38_RodOffersTeddy", 0x3e, 0x43),
		ctx.PlaySFX("Sfx_Step_Soft.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S38_RodOffersTeddy", 0x44, 0x49),
		ctx.PlaySFX("Sfx_Step_Soft.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S38_RodOffersTeddy", 0x4a, 0x4f),
		ctx.PlaySFX("Sfx_Step_Soft.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S38_RodOffersTeddy", 0x50, 0x55),
		ctx.PlaySFX("Sfx_Step_Soft.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S38_RodOffersTeddy", 0x56, -1),
		engine.Immediate(func() {
			if l, ok := ctx.layer("S38_RodOffersTeddy"); ok {
				l.Playing = false
				l.TaskDriven = false
				l.Visible = false
				l.Enabled = false
			}
			loc10SetOriginal(ctx, loc10StateGirl, 0)
		}),
		loc10RestoreRodrigoAndJasmin(ctx),
		ctx.AddItem(10),
		ctx.RemoveItem(12),
		ctx.Say(actor, "038_ROD_12", "[038_ROD_12]"),
	)
}

func loc10TakeMushroom(ctx *Context, actor string) engine.Task {
	if loc10Original(ctx, loc10StateMushroom) == 0 {
		return nil
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1d1, 0xb7, 2), ctx.HideActor(actor), loc10PlayLayerFramesDeferred(ctx, "Gen_RodPickUp", 0, 7), ctx.HideLayer("S39_Pilz"), ctx.DisableArea("S39_Pilz"), ctx.AddItem(8), engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateMushroom, 0) }), loc10PlayLayerFramesDeferred(ctx, "Gen_RodPickUp", 8, -1), ctx.HideLayer("Gen_RodPickUp"), ctx.ShowActor(actor), ctx.Say(actor, "037_ROD_03", "[037_ROD_03]"))
}
func loc10TreeTalk(ctx *Context, actor string) engine.Task {
	if loc10Original(ctx, loc10StateTree) == 0 {
		return loc10WalkVoice(ctx, actor, 0x1a4, 0xda, 6, "040_ROD_01")
	}
	if loc10Original(ctx, loc10StateTreeTalk) == 0 {
		return engine.Sequence(
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateTreeTalk, 1) }),
			loc10WalkVoice(ctx, actor, 0x1a4, 0xda, 6, "040_ROD_01"),
		)
	}
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1b0, 0xdc, 6), loc10TreeStageTalk(ctx, actor))
}

func loc10TreeStageTalk(ctx *Context, actor string) engine.Task {
	switch loc10Original(ctx, loc10StateTreeStage) {
	case 1:
		return engine.Sequence(
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateTreeStage, 2) }),
			ctx.Say(actor, "040_ROD_02", "[040_ROD_02]"),
			ctx.PlayVoiceover("040_JAS_02", "[040_JAS_02]"),
		)
	case 2:
		return engine.Sequence(
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateTreeStage, 3) }),
			ctx.Say(actor, "040_ROD_03", "[040_ROD_03]"),
			ctx.PlayVoiceover("040_JAS_03", "[040_JAS_03]"),
		)
	case 3:
		return engine.Sequence(
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateTreeStage, 4) }),
			ctx.Say(actor, "040_ROD_04", "[040_ROD_04]"),
			ctx.PlayVoiceover("040_JAS_04", "[040_JAS_04]"),
		)
	case 4:
		return loc10ClimbTree(ctx, actor)
	}
	return nil
}

func loc10ClimbTree(ctx *Context, actor string) engine.Task {
	straw := loc10Original(ctx, loc10StateStraw)
	if straw == 3 {
		return engine.Sequence(ctx.Say(actor, "040_ROD_07", "[040_ROD_07]"), ctx.PlayVoiceover("040_JAS_07", "[040_JAS_07]"))
	}
	if straw == 4 {
		return engine.Sequence(ctx.Say(actor, "040_ROD_06", "[040_ROD_06]"), ctx.PlayVoiceover("040_JAS_06", "[040_JAS_06]"))
	}
	tasks := []engine.Task{
		ctx.Say(actor, "040_ROD_05", "[040_ROD_05]"),
		ctx.PlayVoiceover("040_JAS_05", "[040_JAS_05]"),
		ctx.HideActor(actor),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodClimbsTree", 0, 0x21),
		ctx.PlaySFX("Sfx_Clothes_Scratched.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodClimbsTree", 0x22, -1),
		ctx.HideLayer("S40_RodClimbsTree"),
		ctx.ShowActor(actor),
	}
	if straw == 0 || straw == 1 {
		tasks = append(tasks,
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateStraw, 1) }),
			ctx.EnableArea("S40_Root"),
		)
	}
	tasks = append(tasks, ctx.Say(actor, "040_ROD_18", "[040_ROD_18]"))
	return engine.Sequence(tasks...)
}

func loc10RootLook(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1b0, 0xdc, 6),
		ctx.HideActor(actor),
		loc10PlayLayerDeferred(ctx, "S40_RodLooksRoots"),
		ctx.HideLayer("S40_RodLooksRoots"),
		ctx.ShowActor(actor),
		ctx.Say(actor, "040_ROD_18", "[040_ROD_18]"),
	)
}

func loc10TakeNecklace(ctx *Context, actor string) engine.Task {
	if loc10Original(ctx, loc10StateNecklace) == 0 {
		return nil
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x147, 0xfb, 0),
		ctx.HideActor(actor),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodTakesNecklace", 0, 0xc),
		ctx.HideLayer("S40_Necklace"),
		ctx.DisableArea("S40_Necklace"),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodTakesNecklace", 0xd, -1),
		ctx.HideLayer("S40_RodTakesNecklace"),
		ctx.ShowActor(actor),
		ctx.AddItem(16),
		engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateNecklace, 0) }),
		ctx.Say(actor, "040_ROD_10", "[040_ROD_10]"),
	)
}

func loc10TakeStraw(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1b0, 0xdc, 6),
		ctx.HideActor(actor),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodTakesStraw", 0, 8),
		ctx.HideLayer("S40_Straw"),
		ctx.DisableArea("S40_Straw"),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodTakesStraw", 9, -1),
		ctx.HideLayer("S40_RodTakesStraw"),
		engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateStraw, 1) }),
		ctx.AddItem(15),
		ctx.EnableArea("S40_Root"),
		ctx.ShowActor(actor),
		ctx.Say(actor, "040_ROD_09", "[040_ROD_09]"),
	)
}

func loc10DropStraw(ctx *Context, actor string) engine.Task {
	if loc10Original(ctx, loc10StateTree) == 0 || loc10Original(ctx, loc10StateStraw) != 1 {
		return nil
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1b0, 0xdc, 6),
		ctx.HideActor(actor),
		ctx.RemoveItem(15),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodDropsStraw", 0, 0x10),
		ctx.ShowLayer("S40_Straw"),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodDropsStraw", 0x11, -1),
		ctx.HideLayer("S40_RodDropsStraw"),
		engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateStraw, 2) }),
		ctx.MakeLayerClickable("S40_Straw"),
		ctx.DisableArea("S40_Root"),
		ctx.ShowActor(actor),
		ctx.Say(actor, "040_ROD_14", "[040_ROD_14]"),
		ctx.PlayVoiceover("040_JAS_14", "[040_JAS_14]"),
	)
}

func loc10SetFire(ctx *Context, actor string) engine.Task {
	if loc10Original(ctx, loc10StateTree) == 0 || loc10Original(ctx, loc10StateStraw) != 2 {
		return nil
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1b0, 0xdc, 6),
		ctx.Say(actor, "040_ROD_16", "[040_ROD_16]"),
		ctx.PlayVoiceover("040_JAS_16", "[040_JAS_16]"),
		ctx.HideActor(actor),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodSetsFire", 0, 0xc),
		ctx.PlaySFX("Sfx_Match_BeingLit.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodSetsFire", 0xd, -1),
		ctx.HideLayer("S40_RodSetsFire"),
		ctx.ShowActor(actor),
		ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x17c, 0xdc, 6)),
		ctx.HideLayer("S40_Straw"),
		ctx.DisableArea("S40_Straw"),
		loc10PlayLayerFramesDeferred(ctx, "S40_FireStarts", 0, 4),
		ctx.RunAmbient(&loc10S40FireSoundTask{ctx: ctx}),
		loc10PlayLayerFramesDeferred(ctx, "S40_FireStarts", 5, -1),
		ctx.HideLayer("S40_FireStarts"),
		loc10LoopLayer(ctx, "S40_FireLoops"),
		engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateStraw, 3) }),
		ctx.MakeLayerClickable("S40_FireLoops"),
		ctx.PlayVoiceover("040_JAS_17", "[040_JAS_17]"),
		ctx.HideActor(actor),
		ctx.ShowLayer("S40_RodWatchesNecklace"),
		ctx.RunAmbient(loc10PlayLayerDeferred(ctx, "S40_RodWatchesNecklace")),
		loc10PlayLayerFramesDeferred(ctx, "S40_NecklaceIsDropped", 0, 5),
		ctx.PlaySFX("Sfx_Step_Soft.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S40_NecklaceIsDropped", 6, -1),
		ctx.HideLayer("S40_NecklaceIsDropped"),
		ctx.HideLayer("S40_RodWatchesNecklace"),
		ctx.ShowLayer("S40_Necklace"),
		ctx.MakeLayerClickable("S40_Necklace"),
		engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateNecklace, 1) }),
		ctx.ShowActor(actor),
		ctx.SetActorOrientation(actor, 1),
	)
}

func loc10KillFire(ctx *Context, actor string) engine.Task {
	if loc10Original(ctx, loc10StateStraw) != 3 {
		return nil
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x1b0, 0xdc, 6),
		ctx.HideActor(actor),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodKillsFire", 0, 0xc),
		ctx.PlaySFX("Sfx_Step_Soft.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodKillsFire", 0xd, 0x13),
		ctx.PlaySFX("Sfx_Step_Soft.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S40_RodKillsFire", 0x14, -1),
		ctx.HideLayer("S40_RodKillsFire"),
		ctx.ShowActor(actor),
		ctx.HideLayer("S40_FireLoops"),
		ctx.DisableArea("S40_FireLoops"),
		loc10PlayLayerFramesDeferred(ctx, "S40_FireStops", 0, 0xc),
		loc10PlayLayerFramesDeferred(ctx, "S40_FireStops", 0xd, -1),
		ctx.HideLayer("S40_FireStops"),
		ctx.ShowLayer("S40_Straw"),
		ctx.RunAmbient(ctx.WalkToFacingPerspective(actor, 0x17c, 0xdc, 6)),
		engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateStraw, 4) }),
	)
}

type loc10S40FireSoundTask struct {
	ctx         *Context
	handle      audio.Handle
	started     bool
	seenBurning bool
}

func (t *loc10S40FireSoundTask) Update(float64) bool {
	if !t.started {
		sound, err := engine.LoadSound(t.ctx.session.idx, "Sfx_Fire.wav")
		if err != nil {
			return true
		}
		t.handle = t.ctx.session.audioEngine.PlayLooping(sound, audio.CategorySFX)
		t.started = true
	}
	if t.ctx.session.state.Scene != "S40" {
		t.handle.Stop()
		return true
	}
	state := loc10Original(t.ctx, loc10StateStraw)
	if state == 3 {
		t.seenBurning = true
		return false
	}
	if state == 4 || t.seenBurning {
		t.handle.Stop()
		return true
	}
	return false
}

type loc10S40BranchTask struct {
	ctx       *Context
	remaining float64
	current   engine.Task
}

func (t *loc10S40BranchTask) Update(dt float64) bool {
	if t.ctx.session.state.Scene != "S40" || loc10Original(t.ctx, loc10StateTree) == 0 || loc10Original(t.ctx, loc10StateTreeStage) >= 2 {
		return true
	}
	if t.current != nil {
		if !t.current.Update(dt) {
			return false
		}
		t.current = nil
		t.remaining = float64(10 + rand.IntN(11))
		return false
	}
	t.remaining -= dt
	if t.remaining > 0 {
		return false
	}
	if loc10Original(t.ctx, loc10StateTreeTalk) != 0 {
		t.current = engine.Sequence(
			t.ctx.RunAmbient(loc10PlayLayerDeferred(t.ctx, "S40_Branch")),
			t.ctx.PlayVoiceover("040_JAS_01", "[040_JAS_01]"),
		)
		return false
	}
	t.remaining = float64(10 + rand.IntN(11))
	return false
}

func loc10Teddy(ctx *Context, actor string) engine.Task {
	s := loc10Original(ctx, loc10StateTeddy)
	if s == 0 {
		s = 1
	}
	switch s {
	case 1:
		return engine.Sequence(ctx.WalkToFacingPerspective(actor, 0x1b9, 0xd2, 6), ctx.Say(actor, "041_ROD_01", "[041_ROD_01]"), engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateTeddy, 2) }))
	case 2:
		return engine.Sequence(
			ctx.WalkToFacingPerspective(actor, 0x1b9, 0xd2, 6),
			ctx.Say(actor, "041_ROD_02", "[041_ROD_02]"),
			ctx.HideActor(actor),
			loc10HideAllActors(ctx),
			engine.Immediate(func() {
				if l, ok := ctx.layer("S41_RodTakesTeddy"); ok {
					l.Frame = 0
					l.Accumulator = 0
					l.Playing = false
					l.TaskDriven = false
					l.Visible = true
					l.Enabled = true
				}
			}),
			loc10PlayLayerFramesDeferred(ctx, "S41_RodTakesTeddy", 0, -1),
			engine.Immediate(func() {
				if l, ok := ctx.layer("S41_RodTakesTeddy"); ok {
					l.Playing = false
					l.TaskDriven = false
					l.Visible = false
					l.Enabled = false
				}
			}),
			loc10RestoreRodrigoAndJasmin(ctx),
			ctx.AddItem(12),
			engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateTeddy, 3) }),
		)
	default:
		return loc10WalkVoice(ctx, actor, 0x1b9, 0xd2, 6, "041_ROD_01")
	}
}
func loc10TakeLeaf(ctx *Context, actor string) engine.Task {
	if loc10Original(ctx, loc10StateLeaf) == 0 || loc10Original(ctx, loc10StateLeafReady) == 0 {
		return nil
	}
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x143, 0xb8, 4),
		ctx.HideActor(actor),
		loc10PlayLayerFramesDeferred(ctx, "S44_RodTakesLeaf", 0x13, 0x1a),
		ctx.HideLayer("S44_Leaf"),
		ctx.DisableArea("S44_Leaf"),
		engine.Immediate(func() {
			loc10SetOriginal(ctx, loc10StateLeaf, 0)
			loc10SetOriginal(ctx, loc10StateLeafReady, 0)
		}),
		loc10PlayLayerFramesDeferred(ctx, "S44_RodTakesLeaf", 0x1b, -1),
		ctx.HideLayer("S44_RodTakesLeaf"),
		ctx.AddItem(6),
		ctx.ShowActor(actor),
		ctx.Say(actor, "044_ROD_05", "[044_ROD_05]"),
	)
}
func loc10Slug(ctx *Context, actor string) engine.Task {
	return engine.Sequence(
		ctx.WalkToFacingPerspective(actor, 0x125, 0xb8, 4),
		ctx.RemoveItem(8),
		engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateSlug, 3) }),
		loc10PlayLayerFramesDeferred(ctx, "S44_SlugExplodes", 0, 0x2e),
		ctx.PlaySFX("Sfx_Eat.wav"),
		loc10PlayLayerFramesDeferred(ctx, "S44_SlugExplodes", 0x2f, 0x44),
		ctx.HideActor(actor),
		ctx.PlaySFX("Sfx_Explosion_Misc2.wav"),
		engine.Parallel(
			loc10PlayLayerFramesDeferred(ctx, "S44_RodTakesLeaf", 0, 0x12),
			loc10PlayLayerFramesDeferred(ctx, "S44_SlugExplodes", 0x45, -1),
		),
		ctx.HideLayer("S44_SlugExplodes"),
		ctx.ShowLayer("S44_Leaf"),
		ctx.MakeLayerClickable("S44_Leaf"),
		engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateLeafReady, 1) }),
		ctx.HideLayer("S44_RodTakesLeaf"),
		ctx.ShowActor(actor),
		ctx.Say(actor, "044_ROD_03", "[044_ROD_03]"),
	)
}

func loc10WoodChase(ctx *Context) engine.Task {
	return engine.Sequence(loc10PlayLayerFramesDeferred(ctx, "S66_WoodChase", 0, 5), ctx.PlayVoiceover("066_JAS_01", "[066_JAS_01]"), loc10PlayLayerFramesDeferred(ctx, "S66_WoodChase", 6, 0x1d), ctx.PlayVoiceover("066_JAS_02", "[066_JAS_02]"), loc10PlayLayerFramesDeferred(ctx, "S66_WoodChase", 0x1e, 0x28), ctx.PlaySFX("Sfx_Explosion_Misc2.wav"), loc10PlayLayerFramesDeferred(ctx, "S66_WoodChase", 0x29, 0x2f), ctx.PlayVoiceover("066_JAS_03", "[066_JAS_03]"), loc10PlayLayerFramesDeferred(ctx, "S66_WoodChase", 0x30, 0x3d), ctx.PlaySFX("Sfx_Explosion_Misc2.wav"), loc10PlayLayerFramesDeferred(ctx, "S66_WoodChase", 0x3e, -1), engine.Immediate(func() { loc10SetOriginal(ctx, loc10StateGlider, 0) }), ctx.ChangeScene("S37"))
}

func loc10SyncJasminActor(ctx *Context) {
	if ctx.session.scene == nil {
		return
	}
	player := loc10ActorID(ctx)
	jasmin := loc10Original(ctx, loc10StateJas) != 0
	for id, a := range ctx.session.scene.Characters {
		if a == nil {
			continue
		}
		if id == player {
			a.Visible = true
			continue
		}
		name := strings.ToLower(id + " " + a.AssetName)
		a.Visible = jasmin && strings.Contains(name, "jas")
	}
}

func loc10PrepareScene(ctx *Context, scene string) {
	hide := func(ids ...string) {
		for _, id := range ids {
			if l, ok := ctx.layer(id); ok {
				l.Frame = 0
				l.Accumulator = 0
				l.Playing = false
				l.TaskDriven = false
				l.Visible = false
				l.Enabled = false
			}
		}
	}
	show := func(id string) {
		if l, ok := ctx.layer(id); ok {
			l.Frame = 0
			l.Accumulator = 0
			l.Playing = false
			l.TaskDriven = false
			l.Visible = true
			l.Enabled = true
		}
	}
	switch scene {
	case "S37":
		hide("S37_RodFallDown", "S37_JasFallDown", "S37_JasPointN", "S37_GliderArrives", "S37_Glider", "S37_GliderLeaves")
		from := loc10SceneID(ctx.session.state.PreviousScene)
		arrivingByGlider := from == "S113" || ctx.session.state.PreviousLocation == 31
		if loc10Original(ctx, loc10StateGlider) != 0 && !arrivingByGlider {
			show("S37_Glider")
		}
	case "S38":
		hide("S38_RodOffersTeddy", "S38_Girl", "S38_GirlJumps", "S38_RodStealsRope", "S38_JasPointsNW", "Gen_Busch", "Gen_BuschClone")
		if loc10Original(ctx, loc10StateGirl) != 0 {
			show("S38_GirlJumps")
		}
		to57Open := loc10Original(ctx, loc10StateTo57) != 0
		to68Open := loc10Original(ctx, loc10StateStory) != 0
		if !to57Open {
			show("Gen_BuschClone")
		}
		if !to68Open {
			show("Gen_Busch")
		}
		if area, ok := ctx.session.scene.Areas["S38_To57"]; ok {
			area.Enabled = to57Open
		}
		if area, ok := ctx.session.scene.Areas["S38_To68"]; ok {
			area.Enabled = to68Open
		}
		if bg := ctx.session.scene.Background; bg != nil {
			if nav := bg.NavGrid(); nav != nil {
				nav.SetDynamicRect(0xe6, 0x28, 0x1a4, 0xa0, !to57Open)
				nav.SetDynamicRect(0, 0, 0xaa, 0xbe, !to68Open)
			}
		}
	case "S39":
		hide("Gen_Busch", "S39_Pilz", "Gen_RodPickUp")
		if loc10Original(ctx, loc10StateTo99) == 0 {
			show("Gen_Busch")
		}
		if loc10Original(ctx, loc10StateMushroom) != 0 {
			show("S39_Pilz")
		}
	case "S40":
		hide("S40_RodClimbsTree", "S40_RodDropsStraw", "S40_RodKillsFire", "S40_RodLooksRoots", "S40_RodSetsFire", "S40_RodTakesStraw", "S40_RodWatchesNecklace", "S40_RodTakesNecklace", "S40_NecklaceIsDropped", "S40_Necklace", "S40_Straw", "S40_FireStarts", "S40_FireLoops", "S40_FireStops", "S40_Branch")
		show("S40_Branch")
	case "S41":
		hide("S41_RodTakesTeddy")
	case "S42":
		hide("Gen_Busch", "Gen_BuschClone")
		if loc10Original(ctx, loc10StateTo91) == 0 {
			show("Gen_Busch")
			show("Gen_BuschClone")
		}
	case "S44":
		hide("S44_RodTakesLeaf", "S44_SlugExplodes", "S44_Leaf")
		if loc10Original(ctx, loc10StateLeaf) != 0 {
			show("S44_Leaf")
		}
	case "S66":
		hide("S66_WoodChase")
	}
}

func loc10WalkScene(ctx *Context, actor string, x, y float64, d int, scene string) engine.Task {
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, x, y, d), ctx.ChangeScene(scene))
}
func loc10WalkVoice(ctx *Context, actor string, x, y float64, d int, line string) engine.Task {
	return engine.Sequence(ctx.WalkToFacingPerspective(actor, x, y, d), ctx.Say(actor, line, "["+line+"]"))
}
func loc10PlayLayerDeferred(ctx *Context, id string) engine.Task {
	var t engine.Task
	return engine.TaskFunc(func(dt float64) bool {
		if t == nil {
			t = ctx.PlayLayer(id)
		}
		return t.Update(dt)
	})
}
func loc10PlayLayerFramesDeferred(ctx *Context, id string, from, to int) engine.Task {
	var t engine.Task
	return engine.TaskFunc(func(dt float64) bool {
		if t == nil {
			if to < 0 {
				if l, ok := ctx.layer(id); ok {
					to = l.Source.Frames() - 1
				} else {
					return true
				}
			}
			t = ctx.PlayLayerFrames(id, from, to)
		}
		return t.Update(dt)
	})
}
func loc10LoopLayer(ctx *Context, id string) engine.Task {
	return engine.Immediate(func() {
		if l, ok := ctx.layer(id); ok {
			l.Visible = true
			l.Enabled = true
			l.Mode = engine.AnimLoop
			l.Playing = true
			l.TaskDriven = false
		}
	})
}
func loc10SceneID(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if strings.HasPrefix(s, "S") {
		return s
	}
	switch s {
	case "091":
		return "S91"
	case "37", "38", "39", "40", "41", "42", "43", "44", "66", "99", "57", "68", "101", "91", "113":
		return "S" + s
	}
	return s
}
func loc10ActorID(ctx *Context) string {
	for _, name := range []string{"RodrigoLarge", "RodrigoSmall"} {
		if _, id, ok := ctx.session.actorByNameOrAsset(name); ok {
			return id
		}
	}
	if ctx.session.scene != nil && len(ctx.session.scene.CharacterOrder) != 0 {
		return ctx.session.scene.CharacterOrder[0]
	}
	return ""
}

func loc10JasminActorID(ctx *Context) string {
	if ctx.session.scene == nil {
		return ""
	}
	for id, a := range ctx.session.scene.Characters {
		if a == nil {
			continue
		}
		name := strings.ToLower(id + " " + a.AssetName)
		if strings.Contains(name, "jas") {
			return id
		}
	}
	return ""
}

func loc10HideAllActors(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		if ctx.session.scene == nil {
			return
		}
		for _, a := range ctx.session.scene.Characters {
			if a != nil {
				a.Visible = false
			}
		}
	})
}

func loc10RestoreRodrigoAndJasmin(ctx *Context) engine.Task {
	return engine.Immediate(func() {
		if ctx.session.scene == nil {
			return
		}
		rod := loc10ActorID(ctx)
		jas := loc10Original(ctx, loc10StateJas) != 0
		for id, a := range ctx.session.scene.Characters {
			if a == nil {
				continue
			}
			if id == rod {
				a.Visible = true
				continue
			}
			name := strings.ToLower(id + " " + a.AssetName)
			a.Visible = jas && strings.Contains(name, "jas")
		}
	})
}
func loc10Original(ctx *Context, o int) int {
	b := ctx.session.state.OriginalState
	if o < 0 || o+4 > len(b) {
		return 0
	}
	return int(int32(binary.LittleEndian.Uint32(b[o : o+4])))
}
func loc10SetOriginal(ctx *Context, o, v int) {
	b := ctx.session.state.OriginalState
	if o < 0 || o+4 > len(b) {
		return
	}
	binary.LittleEndian.PutUint32(b[o:o+4], uint32(int32(v)))
}
func itoa10(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string([]byte{byte('0' + n/10), byte('0' + n%10)})
}

type loc10GliderIdle struct {
	ctx         *Context
	initialized bool
	baseX       int
	baseY       int
	phase       float64
	accumulator float64
}

func (t *loc10GliderIdle) Update(dt float64) bool {
	if t.ctx.session.state.Location != 10 || t.ctx.session.state.Scene != "S37" || loc10Original(t.ctx, loc10StateGlider) == 0 {
		return true
	}
	layer, ok := t.ctx.layer("S37_Glider")
	if !ok {
		return true
	}
	if !t.initialized {
		t.baseX = layer.X
		t.baseY = layer.Y
		t.initialized = true
	}
	t.accumulator += dt
	for t.accumulator >= 0.04 {
		t.accumulator -= 0.04
		t.phase += 0.1
		if t.phase >= 2*math.Pi {
			t.phase -= 2 * math.Pi
		}
	}
	layer.X = t.baseX
	layer.Y = t.baseY - int(math.Round(math.Sin(t.phase)))
	return false
}

type loc10Ambient struct {
	ctx  *Context
	wait float64
}

func (t *loc10Ambient) Update(dt float64) bool {
	if t.ctx.session.state.Location != 10 {
		return true
	}
	if t.wait <= 0 {
		t.wait = float64(5 + rand.IntN(6))
	}
	t.wait -= dt
	if t.wait > 0 {
		return false
	}
	day := []string{"Sfx_Bird.wav", "Sfx_Bird2.wav", "Sfx_Bird3.wav", "Sfx_Chirp.wav", "Sfx_Fly2.wav"}
	night := []string{"Sfx_Owl.wav", "Sfx_Thunder.wav", "Sfx_Thunder2.wav"}
	a := day
	if loc10Original(t.ctx, loc10StateNight) != 0 {
		a = night
	}
	_ = t.ctx.PlaySFX(a[rand.IntN(len(a))]).Update(0)
	t.wait = float64(5 + rand.IntN(6))
	return false
}
