package game

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"

	"github.com/wok/rent-a-hero/internal/engine"
	"github.com/wok/rent-a-hero/internal/video"
)

const loc02GuestStoryOffset = 0x3ea4

type LOC02Controller struct{}

func (LOC02Controller) Enter(ctx *Context, scene string, from string) engine.Task {
	tasks := []engine.Task{}
	if ctx.session.loadedSave || ctx.session.state.PreviousLocation != 2 {
		tasks = append(tasks, ctx.PlayMusic("Loc02_SanchosPub.wav"))
	}
	switch scene {
	case "S0013":
		tasks = append(tasks, loc02Enter13(ctx, from)...)
	case "S1013":
		tasks = append(tasks, loc02Enter1013(ctx, from)...)
	case "S2013":
		tasks = append(tasks, loc02Enter2013(ctx, from)...)
	case "S0014":
		tasks = append(tasks, loc02Enter14(ctx, from)...)
	case "S0015":
		if !loc02StrangerPresent(ctx) {
			return ctx.ChangeScene("S2013")
		}
		tasks = append(tasks, loc02Enter15(ctx, from)...)
	default:
		return nil
	}
	loc02DisableVideoTransparency(ctx, scene)
	return engine.Sequence(tasks...)
}

func loc02DisableVideoTransparency(ctx *Context, scene string) {
	if scene != "S2013" && scene != "S0015" {
		return
	}
	for _, layer := range ctx.session.scene.Layers {
		if _, ok := layer.Source.(*video.Source); ok {
			layer.ColorKeyed = false
		}
	}
}

func (LOC02Controller) Exit(ctx *Context, scene string, to string) engine.Task { return nil }

func (LOC02Controller) Click(ctx *Context, area string) engine.Task {
	switch ctx.session.state.Scene {
	case "S0013":
		switch area {
		case "S0013_Exit":
			return engine.Sequence(ctx.WalkToFacing(loc02ActorID(ctx), 0x2b, 0x117, 2), ctx.ChangeLocation(3, "S12"))
		case "S0013_To2013":
			return engine.Sequence(ctx.WalkToFacing(loc02ActorID(ctx), 0x1db, 0xf7, 4), ctx.ChangeScene("S2013"))
		case "S0013_To1013":
			return engine.Sequence(ctx.WalkToFacing(loc02ActorID(ctx), 0x189, 0x15e, 0), ctx.ChangeScene("S1013"))
		}
	case "S1013":
		switch area {
		case "S1013_To0013":
			return engine.Sequence(ctx.WalkToFacing(loc02ActorID(ctx), 0x177, 0x164, 7), ctx.ChangeScene("S0013"))
		case "S1013_San":
			return engine.Sequence(ctx.WalkToFacing(loc02ActorID(ctx), 0x1b2, 0xe7, 5), ctx.ChangeScene("S0014"))
		case "S1013_Gast1", "S1013_Gast2":
			return engine.Sequence(ctx.WalkToFacing(loc02ActorID(ctx), 0x136, 0x102, 2), loc02GuestsConversation(ctx))
		}
	case "S2013":
		switch area {
		case "S2013_Fremder":
			if loc02StrangerPresent(ctx) {
				return engine.Sequence(ctx.WalkToFacing(loc02ActorID(ctx), 0x104, 0x163, 0), ctx.ChangeScene("S0015"))
			}
		case "S2013_To0013":
			return engine.Sequence(ctx.WalkToFacing(loc02ActorID(ctx), 0x1a6, 0x117, 6), ctx.ChangeScene("S0013"))
		}
	case "S0014":
		switch area {
		case "S0014_To1013":
			return ctx.ChangeScene("S1013")
		case "S0014_Sancho":
			return loc02SanchoConversation(ctx)
		case "S0014_SanchosOhr":
			return loc02SanchoEar(ctx)
		}
	case "S0015":
		switch area {
		case "S0015_To2013A", "S0015_To2013B":
			return ctx.ChangeScene("S2013")
		case "S0015_Fremder":
			return loc02StrangerConversation(ctx)
		case "S0015_FremderBrust":
			return loc02StrangerChest(ctx)
		case "S0015_Bier":
			return loc02StrangerBeer(ctx)
		}
	}
	return nil
}

func (LOC02Controller) UseItem(ctx *Context, item int, area string) engine.Task { return nil }

func (LOC02Controller) SelectItem(ctx *Context, item int) engine.Task {
	if ctx.session.state.Scene == "S0014" && item == 13 {
		return engine.Sequence(
			ctx.PlayLayerFrames("S0014_SanAnim", 0x118, 0xdc),
			ctx.RemoveItem(13),
			loc02SanchoRodSpeech(ctx, "014_ROD_17"),
			loc02SanchoSpeech(ctx, "014_SAN_17"),
		)
	}
	return nil
}

func (LOC02Controller) LoadConditionMask(ctx *Context, scene string) int {
	if (scene == "S0013" || scene == "S2013") && loc02StrangerPresent(ctx) {
		return 1
	}
	return 0
}

func loc02Enter13(ctx *Context, from string) []engine.Task {
	tasks := []engine.Task{}
	if loc02StrangerPresent(ctx) {
		if _, _, err := ctx.ensureAssetLayer("S0013_Fre"); err == nil {
			tasks = append(tasks, ctx.ShowLayer("S0013_Fre"))
		}
	}
	switch from {
	case "S12":
		tasks = append(tasks, ctx.PlaceActor(loc02ActorID(ctx), 0x2b, 0x117), ctx.SetActorDirection(loc02ActorID(ctx), 6), ctx.ShowActor(loc02ActorID(ctx)))
	case "S1013":
		tasks = append(tasks, ctx.PlaceActor(loc02ActorID(ctx), 0x189, 0x15e), ctx.WalkToFacing(loc02ActorID(ctx), 0x189, 0x159, 4))
	case "S2013":
		if loc02StrangerPresent(ctx) {
			tasks = append(tasks, loc02From2013(ctx))
		}
		tasks = append(tasks, ctx.PlaceActor(loc02ActorID(ctx), 0x1e5, 0xfa), ctx.WalkToFacing(loc02ActorID(ctx), 0x1e5, 0xff, 0))
	default:
		tasks = append(tasks, ctx.PlaceActor(loc02ActorID(ctx), 0x2b, 0x117), ctx.ShowActor(loc02ActorID(ctx)))
	}
	return tasks
}

func loc02Enter1013(ctx *Context, from string) []engine.Task {
	tasks := []engine.Task{
		ctx.MakeLayerClickable("S1013_San"),
		ctx.MakeLayerClickable("S1013_Gast1"),
		ctx.MakeLayerClickable("S1013_Gast2"),
		loc02FreezeIfPresent(ctx, "S1013_San", 0),
		loc02FreezeIfPresent(ctx, "S1013_Gast1", 0),
		loc02FreezeIfPresent(ctx, "S1013_Gast2", 0),
		ctx.RunAmbient(newLoc02SanchoAmbient(ctx)),
	}
	if ctx.GetFlag(FlagLoc05PirateAttackSeen) == 0 || loc02Original(ctx, loc02GuestStoryOffset) == 3 {
		tasks = append(tasks, ctx.RunAmbient(newLoc02GuestAmbient(ctx)))
	}
	switch from {
	case "S0013":
		tasks = append(tasks, ctx.PlaceActor(loc02ActorID(ctx), 0x177, 0x164), ctx.ShowActor(loc02ActorID(ctx)))
	case "S0014":
		tasks = append(tasks, ctx.PlaceActor(loc02ActorID(ctx), 0x1ab, 0xe8), ctx.WalkToFacing(loc02ActorID(ctx), 0x15f, 0xf4, 1))
	default:
		tasks = append(tasks, ctx.ShowActor(loc02ActorID(ctx)))
	}
	return tasks
}

func loc02Enter2013(ctx *Context, from string) []engine.Task {
	tasks := []engine.Task{ctx.EnableArea("S2013_To0013")}
	if loc02StrangerPresent(ctx) {
		loc02UseBackgroundSource(ctx, "S2013_Back_mf")
		tasks = append(tasks, ctx.EnableArea("S2013_Fremder"))
	} else {
		tasks = append(tasks, ctx.DisableArea("S2013_Fremder"))
	}
	switch from {
	case "S0013":
		if loc02StrangerPresent(ctx) {
			tasks = append(tasks, loc02From0013(ctx))
		}
		tasks = append(tasks, ctx.PlaceActor(loc02ActorID(ctx), 0x18a, 0x10e), ctx.WalkToFacing(loc02ActorID(ctx), 0x17c, 0x118, 0))
	case "S0015":
		tasks = append(tasks, ctx.PlaceActor(loc02ActorID(ctx), 0x104, 0x163), ctx.WalkToFacing(loc02ActorID(ctx), 0x127, 0x12d, 5))
	default:
		tasks = append(tasks, ctx.ShowActor(loc02ActorID(ctx)))
	}
	return tasks
}

func loc02Enter14(ctx *Context, from string) []engine.Task {
	tasks := []engine.Task{
		ctx.EnableArea("S0014_To1013"),
		ctx.EnableArea("S0014_Sancho"),
		ctx.EnableArea("S0014_SanchosOhr"),
		loc02FreezeIfPresent(ctx, "S0014_SanAnim", 0),
	}
	if loc02Original(ctx, 0x3ec0) != 0 {
		loc02SetOriginal(ctx, 0x3ec0, 0)
		tasks = append(tasks, loc02SanchoSpeech(ctx, "014_SAN_01"))
	}
	return tasks
}

func loc02Enter15(ctx *Context, from string) []engine.Task {
	for _, id := range []string{"S0015_FreNeck", "S0015_FreBum", "S0015_FreMesser", "S0015_FrePack", "S0015_Piraten", "S0015_RodWunder"} {
		_, _, _ = ctx.ensureAssetLayer(id)
	}
	tasks := []engine.Task{
		ctx.EnableArea("S0015_To2013A"),
		ctx.EnableArea("S0015_To2013B"),
		ctx.EnableArea("S0015_Fremder"),
		loc02FreezeIfPresent(ctx, "S0015_FreDrink", 0),
		loc02FreezeIfPresent(ctx, "S0015_FreTalk", 0),
		loc02FreezeIfPresent(ctx, "S0015_RodTalk", 0),
	}
	if loc02Original(ctx, 0x3ebc) < 3 {
		tasks = append(tasks, ctx.EnableArea("S0015_FremderBrust"))
	}
	if loc02Original(ctx, 0x3eb8) < 3 {
		tasks = append(tasks, ctx.EnableArea("S0015_Bier"))
	}
	return tasks
}

func loc02SanchoRodSpeech(ctx *Context, line string) engine.Task {
	return loc02LayerSpeech(ctx, "S0014_SanAnim", line, 0xb9, 0xc1)
}

func loc02SanchoSpeech(ctx *Context, line string) engine.Task {
	return loc02LayerSpeech(ctx, "S0014_SanAnim", line, 0x96, 0x9e)
}

func loc02SanchoConversation(ctx *Context) engine.Task {
	a := loc02Original(ctx, 0x3ea8)
	b := loc02Original(ctx, 0x3eac)
	if a == 1 || b == 1 {
		loc02SetOriginal(ctx, 0x3ea8, 2)
		loc02SetOriginal(ctx, 0x3eac, 2)
		return engine.Sequence(loc02SanchoRodSpeech(ctx, "014_ROD_01"), loc02SanchoSpeech(ctx, "014_SAN_02"))
	}
	if loc02Original(ctx, 0x3ea0) == 0 {
		switch a {
		case 2:
			loc02SetOriginal(ctx, 0x3ea8, 3)
			return engine.Sequence(loc02SanchoRodSpeech(ctx, "014_ROD_02"), loc02SanchoSpeech(ctx, "014_SAN_03"))
		case 3:
			loc02SetOriginal(ctx, 0x3ea8, 4)
			return engine.Sequence(loc02SanchoRodSpeech(ctx, "014_ROD_03"), loc02SanchoSpeech(ctx, "014_SAN_04"))
		case 4:
			loc02SetOriginal(ctx, 0x3ea8, 5)
			return engine.Sequence(
				loc02SanchoRodSpeech(ctx, "014_ROD_04"),
				ctx.PlayLayerFrames("S0014_SanAnim", 0x9e, 0xaa),
				ctx.PlaySpeechBoundToLayer("S0014_SanAnim", "014_SAN_05", "[014_SAN_05]", 0xaa, 0xb2),
				ctx.PlayLayerFrames("S0014_SanAnim", 0xb2, 0xb9),
				ctx.FreezeLayer("S0014_SanAnim", 0),
			)
		case 5:
			loc02SetOriginal(ctx, 0x3ea8, 6)
			return engine.Sequence(
				ctx.PlayLayerFrames("S0014_SanAnim", 0xc1, 0xcd),
				ctx.PlaySpeechBoundToLayer("S0014_SanAnim", "014_ROD_05", "[014_ROD_05]", 0xcd, 0xd5),
				ctx.PlayLayerFrames("S0014_SanAnim", 0xd5, 0xdc),
				ctx.PlayLayerFrames("S0014_SanAnim", 0x9e, 0xaa),
				ctx.PlaySpeechBoundToLayer("S0014_SanAnim", "014_SAN_06", "[014_SAN_06]", 0xaa, 0xb2),
				ctx.PlayLayerFrames("S0014_SanAnim", 0xb2, 0xb9),
				loc02SanchoRodSpeech(ctx, "014_ROD_06"),
			)
		case 6:
			if loc02Original(ctx, 0x3ec4) != 0 {
				loc02SetOriginal(ctx, 0x3ea8, 7)
			}
			return engine.Sequence(loc02SanchoRodSpeech(ctx, "014_ROD_07"), loc02SanchoSpeech(ctx, "014_SAN_07"))
		case 7:
			if !ctx.session.state.HasItem(13) {
				loc02SetOriginal(ctx, 0x3ea8, 8)
				return engine.Sequence(
					loc02SanchoRodSpeech(ctx, "014_ROD_08"), loc02SanchoSpeech(ctx, "014_SAN_08"),
					loc02SanchoRodSpeech(ctx, "014_ROD_09"), loc02SanchoSpeech(ctx, "014_SAN_09"),
				)
			}
			return engine.Sequence(loc02SanchoRodSpeech(ctx, "014_ROD_07"), loc02SanchoSpeech(ctx, "014_SAN_07"))
		case 8:
			loc02SetOriginal(ctx, 0x3ea8, 7)
			return engine.Sequence(
				loc02SanchoRodSpeech(ctx, "014_ROD_10"), loc02SanchoSpeech(ctx, "014_SAN_10"),
				ctx.PlayLayerFrames("S0014_SanAnim", 0xdc, 0x118),
				ctx.AddItem(13),
			)
		}
		return engine.Sequence(loc02SanchoRodSpeech(ctx, "014_ROD_07"), loc02SanchoSpeech(ctx, "014_SAN_07"))
	}
	if b == 2 {
		loc02SetOriginal(ctx, 0x3eac, 3)
		return engine.Sequence(
			loc02SanchoRodSpeech(ctx, "014_ROD_11"), loc02SanchoSpeech(ctx, "014_SAN_11"),
			loc02SanchoRodSpeech(ctx, "014_ROD_12"), loc02SanchoSpeech(ctx, "014_SAN_12"),
		)
	}
	if b == 3 {
		return engine.Sequence(loc02SanchoRodSpeech(ctx, "014_ROD_13"), loc02SanchoSpeech(ctx, "014_SAN_13"))
	}
	return nil
}

func loc02SanchoEar(ctx *Context) engine.Task {
	count := loc02Original(ctx, 0x3eb0)
	if count > 0 && count < 3 {
		loc02SetOriginal(ctx, 0x3eb0, count+1)
		return engine.Sequence(
			ctx.PlayLayerFrames("S0014_SanAnim", 0, 0x15), ctx.PlaySFX("Sfx_Boing.wav"),
			ctx.PlayLayerFrames("S0014_SanAnim", 0x16, 0x37), ctx.PlaySFX("Sfx_Boioing.wav"),
			ctx.PlayLayerFrames("S0014_SanAnim", 0x38, 0x4d), ctx.PlaySFX("Sfx_Knock_Wet.wav"),
			ctx.PlayLayerFrames("S0014_SanAnim", 0x4e, 0x4f), ctx.PlaySFX("Sfx_Tock.wav"),
			ctx.PlayLayerFrames("S0014_SanAnim", 0x50, 0x6c), ctx.PlaySFX("Sfx_Nose_BeingBroken.wav"),
			ctx.PlayLayerFrames("S0014_SanAnim", 0x6d, 0x81), ctx.PlaySFX("Sfx_Knock_Wet.wav"),
			ctx.PlayLayerFrames("S0014_SanAnim", 0x82, 0x96),
			loc02SanchoRodSpeech(ctx, "014_ROD_14"), loc02SanchoSpeech(ctx, "014_SAN_14"), loc02SanchoRodSpeech(ctx, "014_ROD_15"),
		)
	}
	if count == 3 {
		return engine.Sequence(
			ctx.PlayLayerFrames("S0014_SanAnim", 0x78, 0x7b), ctx.Wait(0.2), ctx.PlayLayerFrames("S0014_SanAnim", 0x7b, 0x78),
			loc02SanchoSpeech(ctx, "014_SAN_16"),
		)
	}
	return nil
}

func loc02StrangerConversation(ctx *Context) engine.Task {
	stage := loc02Original(ctx, 0x3eb4)
	switch stage {
	case 1:
		loc02SetOriginal(ctx, 0x3eb4, 2)
		return loc02StrangerExchange(ctx, "015_ROD_02", "015_FRE_02")
	case 2:
		loc02SetOriginal(ctx, 0x3eb4, 3)
		return loc02StrangerExchange(ctx, "015_ROD_03", "015_FRE_03")
	case 3:
		loc02SetOriginal(ctx, 0x3eb4, 4)
		return loc02StrangerExchange(ctx, "015_ROD_04", "015_FRE_04")
	case 4:
		return loc02StrangerFinale(ctx)
	}
	return nil
}

func loc02StrangerFinale(ctx *Context) engine.Task {
	return engine.Sequence(
		ctx.PlaySpeechBoundToLayer("S0015_RodTalk", "015_ROD_05", "[015_ROD_05]", 0, 8),
		ctx.FreezeLayer("S0015_RodTalk", 0),
		ctx.RunAmbient(ctx.PlayVoiceover("015_FRE_05", "[015_FRE_05]")),
		ctx.ShowLayer("S0015_FrePack"),
		ctx.PlayLayerFrames("S0015_FrePack", 0, 0x2d),
		ctx.PlaySFX("Sfx_Eat.wav"),
		ctx.RunAmbient(ctx.PlayVoiceover("015_ROD_06", "[015_ROD_06]")),
		ctx.PlayLayerFrames("S0015_FrePack", 0x2e, 0x36),
		ctx.PlaySFX("Sfx_Eat.wav"),
		ctx.PlayLayerFrames("S0015_FrePack", 0x37, 0x3c),
		ctx.RunAmbient(ctx.PlayVoiceover("015_FRE_06", "[015_FRE_06]")),
		ctx.PlayLayerFrames("S0015_FrePack", 0x3d, 0x45),
		ctx.PlaySFX("Sfx_DangerStringsLow.wav"),
		ctx.PlayLayerFrames("S0015_FrePack", 0x46, -1),
		ctx.HideLayer("S0015_FrePack"),
		ctx.ShowLayer("S0015_Piraten"),
		ctx.PlayLayerFrames("S0015_Piraten", 0, 0x0e),
		ctx.PlaySFX("Sfx_DangerStrings.wav"),
		ctx.RunAmbient(ctx.PlayVoiceover("015_ST1_01", "[015_ST1_01]")),
		ctx.PlayLayerFrames("S0015_Piraten", 0x0f, -1),
		ctx.HideLayer("S0015_Piraten"),
		ctx.ShowLayer("S0015_RodWunder"),
		ctx.PlaySFX("Sfx_DangerStringsHigh.wav"),
		ctx.PlayLayerFrames("S0015_RodWunder", 0, -1),
		ctx.HideLayer("S0015_RodWunder"),
		engine.Immediate(func() {
			loc02SetOriginal(ctx, 0x3ea0, 0)
			ctx.SetFlag(flagLoc05QuestB, 0)
			loc02SetOriginal(ctx, 0x3eb4, 5)
			loc02SetOriginal(ctx, 0x3ecc, 1)
			ctx.SetFlag(FlagLoc03RaidersGone, 1)
		}),
		ctx.AddItem(1),
		ctx.ChangeLocation(3, "S12"),
	)
}

func loc02StrangerExchange(ctx *Context, rod, fre string) engine.Task {
	return engine.Sequence(
		ctx.PlaySpeechBoundToLayer("S0015_RodTalk", rod, "["+rod+"]", 0, 8),
		ctx.FreezeLayer("S0015_RodTalk", 0),
		ctx.PlaySpeechBoundToLayer("S0015_FreTalk", fre, "["+fre+"]", 0, 8),
		ctx.FreezeLayer("S0015_FreTalk", 0),
	)
}

func loc02StrangerChest(ctx *Context) engine.Task {
	stage := loc02Original(ctx, 0x3ebc)
	if stage >= 3 {
		return nil
	}
	loc02SetOriginal(ctx, 0x3ebc, stage+1)
	return engine.Sequence(
		ctx.ShowLayer("S0015_FreNeck"),
		ctx.PlayLayerFrames("S0015_FreNeck", 0, 0xc), ctx.PlaySFX("Sfx_Boing2.wav"),
		ctx.PlayLayerFrames("S0015_FreNeck", 0xd, -1), ctx.HideLayer("S0015_FreNeck"),
	)
}

func loc02StrangerBeer(ctx *Context) engine.Task {
	stage := loc02Original(ctx, 0x3eb8)
	if stage == 1 {
		loc02SetOriginal(ctx, 0x3eb8, 2)
		return engine.Sequence(
			ctx.ShowLayer("S0015_FreBum"), ctx.PlayLayerFrames("S0015_FreBum", 0, 0x10), ctx.PlaySFX("Sfx_TableHit.wav"),
			ctx.PlayLayerFrames("S0015_FreBum", 0x11, -1), ctx.HideLayer("S0015_FreBum"),
		)
	}
	if stage == 2 {
		loc02SetOriginal(ctx, 0x3eb8, 3)
		return engine.Sequence(
			ctx.ShowLayer("S0015_FreMesser"), ctx.PlayLayerFrames("S0015_FreMesser", 0, 0x13), ctx.PlaySFX("Sfx_Woosh.wav"),
			ctx.PlayLayerFrames("S0015_FreMesser", 0x14, -1), ctx.HideLayer("S0015_FreMesser"),
		)
	}
	return nil
}

func loc02From2013(ctx *Context) engine.Task {
	if _, _, err := ctx.ensureAssetLayer("S0013_From2013"); err != nil {
		return engine.Immediate(func() {})
	}
	return engine.Sequence(
		ctx.HideActor(loc02ActorID(ctx)),
		ctx.ShowLayer("S0013_From2013"),
		ctx.PlayLayerFrames("S0013_From2013", 0, 6),
		ctx.PlaySFX("Gen_StepLeft.wav"),
		ctx.PlayLayerFrames("S0013_From2013", 7, 11),
		ctx.PlaySFX("Gen_StepRight.wav"),
		ctx.PlayLayerFrames("S0013_From2013", 12, 16),
		ctx.PlaySFX("Gen_StepLeft.wav"),
		ctx.PlayLayerFrames("S0013_From2013", 17, 21),
		ctx.PlaySFX("Gen_StepRight.wav"),
		ctx.PlayLayerFrames("S0013_From2013", 22, 27),
		ctx.PlaySFX("Gen_StepLeft.wav"),
		ctx.PlayLayerFrames("S0013_From2013", 28, -1),
		ctx.HideLayer("S0013_From2013"),
		ctx.ShowActor(loc02ActorID(ctx)),
	)
}

func loc02From0013(ctx *Context) engine.Task {
	if _, _, err := ctx.ensureAssetLayer("S2013_From0013"); err != nil {
		return engine.Immediate(func() {})
	}
	return engine.Sequence(
		ctx.HideActor(loc02ActorID(ctx)),
		ctx.ShowLayer("S2013_From0013"),
		ctx.PlayLayerFrames("S2013_From0013", 0, 3),
		ctx.PlaySFX("Gen_StepLeft.wav"),
		ctx.PlayLayerFrames("S2013_From0013", 4, 10),
		ctx.PlaySFX("Gen_StepRight.wav"),
		ctx.PlayLayerFrames("S2013_From0013", 11, 17),
		ctx.PlaySFX("Gen_StepLeft.wav"),
		ctx.PlayLayerFrames("S2013_From0013", 18, 24),
		ctx.PlaySFX("Gen_StepRight.wav"),
		ctx.PlayLayerFrames("S2013_From0013", 25, -1),
		ctx.PlaySFX("Gen_StepRight.wav"),
		ctx.HideLayer("S2013_From0013"),
		ctx.ShowActor(loc02ActorID(ctx)),
	)
}

func loc02GuestsConversation(ctx *Context) engine.Task {
	story := loc02Original(ctx, loc02GuestStoryOffset)
	if ctx.GetFlag(FlagLoc05PirateAttackSeen) == 0 || story == 3 {
		line := fmt.Sprintf("013_ROD_%02d", 1+rand.IntN(4))
		return ctx.Say(loc02ActorID(ctx), line, "["+line+"]")
	}
	if story == 1 {
		loc02SetOriginal(ctx, loc02GuestStoryOffset, 2)
		return engine.Sequence(
			ctx.Say(loc02ActorID(ctx), "013_ROD_05", "[013_ROD_05]"),
			loc02LayerSpeech(ctx, "S1013_Gast1", "013_GA1_01", 0, 5),
			loc02LayerSpeech(ctx, "S1013_Gast2", "013_GA2_01", 0, 4),
			ctx.Say(loc02ActorID(ctx), "013_ROD_06", "[013_ROD_06]"),
			loc02LayerSpeech(ctx, "S1013_Gast1", "013_GA1_02", 0, 5),
		)
	}
	if story == 2 {
		loc02SetOriginal(ctx, loc02GuestStoryOffset, 3)
		return engine.Sequence(
			ctx.Say(loc02ActorID(ctx), "013_ROD_07", "[013_ROD_07]"),
			loc02LayerSpeech(ctx, "S1013_Gast1", "013_GA1_03", 0, 5),
			loc02LayerSpeech(ctx, "S1013_Gast1", "013_GA1_04", 0, 5),
			ctx.Wait(2),
			loc02LayerSpeech(ctx, "S1013_Gast1", "013_GA1_05", 0, 5),
			ctx.PlayLayerFrames("S1013_Gast2", 18, 19),
			ctx.PlaySFX("Sfx_Knock_Wet.wav"),
			ctx.PlayLayerFrames("S1013_Gast2", 20, 23),
			ctx.Wait(2.5),
			ctx.PlayLayerFrames("S1013_Gast2", 20, 18),
			loc02LayerSpeech(ctx, "S1013_Gast2", "013_GA2_02", 4, 12),
		)
	}
	return ctx.Say(loc02ActorID(ctx), "013_ROD_01", "[013_ROD_01]")
}

func loc02LayerSpeech(ctx *Context, layer, line string, start, end int) engine.Task {
	if l, ok := ctx.session.scene.Layers[layer]; ok && l.Source != nil {
		return engine.Sequence(ctx.ShowLayer(layer), ctx.PlaySpeechBoundToLayer(layer, line, "["+line+"]", start, end), ctx.FreezeLayer(layer, 0))
	}
	return ctx.PlayVoiceover(line, "["+line+"]")
}

func loc02FreezeIfPresent(ctx *Context, layer string, frame int) engine.Task {
	if _, ok := ctx.session.scene.Layers[layer]; !ok {
		return engine.Immediate(func() {})
	}
	return engine.Sequence(ctx.ShowLayer(layer), ctx.FreezeLayer(layer, frame))
}

func loc02ActorID(ctx *Context) string {
	if actor := ctx.session.PlayerActor(); actor != nil {
		return actor.ID
	}
	return "RodrigoSmall"
}

func loc02StrangerPresent(ctx *Context) bool {
	return ctx.GetFlag(flagLoc05QuestB) != 0
}

func loc02UseBackgroundSource(ctx *Context, stem string) {
	if ctx.session.scene == nil || ctx.session.scene.Background == nil {
		return
	}
	source, _, err := engine.LoadSpriteSourceByStem(ctx.session.idx, stem, ".bmp", ".tcc", ".avi", ".a16")
	if err != nil {
		return
	}
	ctx.session.scene.Background.Source = source
}

func loc02Original(ctx *Context, offset int) int {
	state := ctx.session.state.OriginalState
	if offset < 0 || offset+4 > len(state) {
		return 0
	}
	return int(int32(binary.LittleEndian.Uint32(state[offset : offset+4])))
}

func loc02SetOriginal(ctx *Context, offset int, value int) {
	state := ctx.session.state.OriginalState
	if offset < 0 || offset+4 > len(state) {
		return
	}
	binary.LittleEndian.PutUint32(state[offset:offset+4], uint32(int32(value)))
}

type loc02Ambient struct {
	ctx       *Context
	remaining float64
	active    engine.Task
	guest     bool
}

func newLoc02SanchoAmbient(ctx *Context) engine.Task {
	return &loc02Ambient{ctx: ctx, remaining: float64(10 + rand.IntN(11))}
}

func newLoc02GuestAmbient(ctx *Context) engine.Task {
	return &loc02Ambient{ctx: ctx, remaining: float64(10+rand.IntN(21)) / 10, guest: true}
}

func (t *loc02Ambient) Update(dt float64) bool {
	if t.ctx.session.state.Scene != "S1013" {
		return true
	}
	if t.active != nil {
		if !t.active.Update(dt) {
			return false
		}
		t.active = nil
		if t.guest {
			t.remaining = float64(5 + rand.IntN(11))
		} else {
			t.remaining = float64(5 + rand.IntN(11))
		}
		return false
	}
	if t.ctx.session.Locked() {
		return false
	}
	t.remaining -= dt
	if t.remaining > 0 {
		return false
	}
	if t.guest {
		t.active = engine.Sequence(
			t.ctx.PlayLayerFrames("S1013_Gast1", 38, 58),
			t.ctx.PlaySFX("Sfx_Gluck.wav"),
			t.ctx.PlayLayerFrames("S1013_Gast1", 59, 63),
			t.ctx.PlaySFX("Sfx_Tock.wav"),
			t.ctx.PlayLayerFrames("S1013_Gast1", 64, 70),
			t.ctx.FreezeLayer("S1013_Gast1", 38),
		)
	} else {
		t.active = engine.Sequence(t.ctx.PlayLayerFrames("S1013_San", 12, -1), t.ctx.FreezeLayer("S1013_San", 0))
	}
	return false
}
