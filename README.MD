# go-rent-a-hero

This is a Go based rewrite of the game engine for [Rent-a-Hero (1998)](https://en.wikipedia.org/wiki/Rent-a-Hero_(1998_video_game)
) by neo Software. It was achieved by reverse-engineering the assembly code of the old 1998 executable.

## Dependencies

### The game

You need a copy of this game. Either:

- A physical CD
- A copy of the contents of the CD
- An image of the CD

### Native libraries

There's exactly one native dependency:

- SDL3

The *Releases* section of this project contains pre-built versions of this library for macOS and Windows. For other platforms you might have to compile it yourself.

## Installation

- If you downloaded the pre-built macOS build, run `xattr -cr .` from the directory containing `rah` and `rahinstall`
- Run the command line tool `rahinstall` to prepare the `data` directory. It will unpack the InstallShield cabinet files and copy common data from the CD.
  
  ```
  ./rahinstall RENT-A-HERO.ISO data/
  ```

## Building from source

If you wish to build/run this directly from source, you may use:

- ```
  go run ./cmd/rahinstall RENTAHERO.ISO data
  ```
- and then
- ```
  go run cmd/rah/main.go  
  ```
