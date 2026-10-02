package game

// LaunchPhase is a stage of the real launch sequence (Location 40's intro
// cinematic, Location 35's interactive interlude, then normal gameplay).
// See ResolveLaunchPhase.
type LaunchPhase int

const (
	// PhaseIntroPart1 is Location 40, Scene 114 (cinematic; see
	// LoadIntroPart1Task).
	PhaseIntroPart1 LaunchPhase = iota
	// PhaseLoc35 is Location 35, Scene "S1" (interactive; see
	// LOC35Controller).
	PhaseLoc35
	// PhaseIntro115 is Location 40, Scene 115 (cinematic; see
	// LoadIntroPart115Task).
	PhaseIntro115
	// PhaseLoc37 is Location 37, Scene S3 (interactive; see
	// LOC37Controller and agents/scenes/3.MD).
	PhaseLoc37
	// PhaseIntro116 is Location 40, Scene 116 (cinematic; see
	// LoadIntroPart116Task).
	PhaseIntro116
	// PhaseGameplay is everything else: the caller's own --location/idx and
	// Controller (LOC01 by default).
	PhaseGameplay
)

// ResolveLaunchPhase reports which phase of the real launch sequence
// sceneID belongs to, per agents/scenes/01_INTRO.MD (Location 40's Scenes
// 114/115/116) and agents/scenes/02_DRAGON_BLASTER_DELUXE.MD (Location
// 35's Scene S1) -- so a debug entry point (config.ini's init_scene, or
// --scene) can name any known scene, cinematic or not, and have the engine
// figure out on its own which location it lives under and which earlier
// phases to skip, rather than the caller having to separately pass
// --location and manually know, say, that Scene 116 lives mid-cinematic or
// that Scene S1 is a whole different location from the one --location
// names.
//
// Every phase before the one sceneID resolves to is skipped entirely: e.g.
// init_scene=115 jumps into Scene 115 (skipping Scene 114 and Location 35),
// then runs Location 37 / Scene S3 before Scene 116; init_scene=116 skips
// straight to Scene 116; init_scene=S3 skips to the Location 37 interlude;
// and init_scene=46 (an
// ordinary LOC01 scene, PhaseGameplay's default) skips the entire launch
// cinematic and drops the player directly into gameplay there.
//
// A sceneID this package doesn't specifically know about -- i.e. every
// ordinary gameplay scene -- resolves to PhaseGameplay, unchanged from
// before this existed: the caller's own --location/idx and initial scene.
func ResolveLaunchPhase(sceneID string) LaunchPhase {
	switch sceneID {
	case "114":
		return PhaseIntroPart1
	case loc35SceneID:
		return PhaseLoc35
	case "115":
		return PhaseIntro115
	case loc37SceneID, "3":
		return PhaseLoc37
	case "116":
		return PhaseIntro116
	default:
		return PhaseGameplay
	}
}
