# Play a metronome click
A command-line metronome.

Of course there are much better metronome apps for phones these days. My main use case for writing this was to have an option (-mrate) to automatically increase the tempo of the metronome to practice parts at increasing speeds. I also have a foot pedal that simulates key presses. I wanted to use it to increase/decrease the rate manually (-tempo-keys).

## Usage
```text
Usage of metronome:
  -c int
        count in this many measures (default 1)
  -mrate int
        increase rate every 'mrate' measures up to a rate of 200 max
  -tempo int
        specify the tempo in beats per minute (default 60)
  -tempo-keys string
        increase or decrease tempo using keyboard keys. Example: "12" increases tempo when 1 is pressed and decreases tempo when 2 is pressed (default ", ")
  -tsig int
        specify the beats per measure (default 4)
  -vol float
        increase/decrease volume logarithmically, pos or neg. Try using 1.3 as the value.
```

## References
* afplay <file> - play a sound file from MacOS command line
* github.com/faiface/beep
* [Click samples](https://stash.reaper.fm/40824/Metronomes.zip)