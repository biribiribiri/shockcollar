[![GoDoc](https://godoc.org/github.com/biribiribiri/shockcollar?status.svg)](http://godoc.org/github.com/biribiribiri/shockcollar)

# shockcollar
Raspberry Pi based remote for the Sportdog SD-400 shock collar in Go. It uses
rpitx to transmit RF directly to the GPIO. You *only* need a Raspberry Pi, no
other equipment needed.

[Web UI Demo](http://htmlpreview.github.io/?https://github.com/biribiribiri/shockcollar/blob/master/remote/static/index.html)

## Disclaimers

This software uses [rpitx](https://github.com/F5OEO/rpitx) to transmit RF
signals directly from the Raspberry Pi's GPIO pins. Before you transmit, know
your laws. You are responsible for using this software legally.

This software is really janky. I would strongly discourage you from using this
for your production shock collar service. rpitx has also been reported to
cause file system corruption in the past, so I would recommend backing up your
Raspberry Pi before you attempt to use this software.

This project is not associated with Sportdog.

## Getting Started

Plug a wire on [GPIO 4 (Pin 7)](https://pinout.xyz/pinout/pin7_gpio4) on the
GPIO header. This will serve as the antenna.

Then on your Raspberry Pi:
```
git clone https://github.com/F5OEO/rpitx.git
cd rpitx
# This is the version of rpitx I've tested with. Newer versions may or may not work.
git checkout 1423af8c6eb201a92d992965e01073717cbc5fdb
./install.sh

go get github.com/biribiribiri/shockcollar
cd $GOPATH/src/github.com/biribiribiri/shockcollar
go get ./...
cd $GOPATH/src/github.com/biribiribiri/shockcollar/remote

# Start the server. Collar commands can be sent via the CLI, gRPC, or via HTTP.
# Needs to be run as root because rpitx needs to be run as root.
# Probably not super secure...
sudo ./remote --rpitx=$PATH_TO_RPITX
```

After starting the remote, you need to pair your collar:

 1. Turn the collar off.
 2. Press and hold the collar power button until the collar LED turns off
    (4-5 seconds)
 3. Send the "continuous stimulation" (shock 1 8s) or "beep" (beep 8s) command
    from the remote until the collar LED blinks 5 times.

A simple web interface will start running on port 8000. You can use this to
send commands to the collar.

# Reverse Engineering Details

The Sportdog Fieldtrainer 400 (SD-400) is a shock collar with momentary
stimulation, continuous stimulation, and tone capabilities. The collar model
number is FR-200, and the remote model number is FT-100.

This documentation should apply to any Sportdog training collar compatible
with the FT-100 remote (including SD-400, SD-400S and SD-400CAMO).

## Communication Protocol

The SD-400 remote uses binary FSK (frequency-shift keying) to send commands to
the collar.

The carrier frequency is 27.255 MHz (Citizens Band Channel 23).

This document uses the following symbol definitions:
'0': 0 Hz (unmodulated), 4 ms duration
'1': 5 kHz modulation, 4 ms duration
'2': 5 kHz modulation, 2 ms duration

The 2ms '2' symbol only appears to demarcate the end of a portion of a
continuous command. All data symbols are either '0' or '1'.

## Commands

There are two classes of commands sent by the remote:

1. Momentary: Used for the momentary stimulation (nick) function.
2. Continuous: Used for continuous stimulation and tone function.

### Momentary Commands

"Momentary" commands sent by the remote are 43-bits long (172 ms).

```
Format:
  1 1111 0001 [16-bit remote ID] [8-bit command type] [8-bit command arg] 10
```

### Continuous Commands

"Continuous" commands have a similar structure to the momentary commands,
but the transmission sequence repeats for as long as the button is held.

```
Start:
  1 1111 0001 [16-bit remote ID] [8-bit command type] [8-bit command arg] 102
Repeated N times:
         0001 [16-bit remote ID] [8-bit command type] [8-bit command arg] 102
End:
         0001 [16-bit remote ID] [8-bit command type] [8-bit command arg] 1021111111111
```

For both momentary and continuous commands, if the remote is in "standby-mode"
(no buttons pressed for some time), then the initial sequence of 5 '1's
(20ms), becomes 23.5 bits long instead (94ms). However, the collar doesn't
seem to require this.

The FT-100 has an 8-position dial, and 3 buttons (up, down, side). In the
default Mode 1, these buttons have the following functions:
* Up: Continuous shock at the specified level for up to 8 seconds.
* Down: Nick at the specified level.
* Side: Beep.

At this time, only the Mode 1 commands have been documented. The other modes
provide additional functionality, such as exposing 16 levels of continuous
stimulation.

## Remote IDs

Example remote IDs: 0x6695, 0x999a

The remote is TX-only and the collar is RX-only. A remote has a fixed ID
that it broadcasts. A collar will pair to a remote, and will only listen to
its paired remote's ID. To pair a collar to a real or emulated remote:

 1. Turn the collar off.
 2. Press and hold the collar power button until the collar LED turns off
    (4-5 seconds)
 3. Send the "continuous stimulation" or "beep" command from the remote
    until the collar LED blinks 5 times.

## Command and Argument Values

### Command Type: Beep (0x59)

 * Arg: 0xa9

### Command Type: Nick (0x6a)

 * Level 1: 0xa9
 * Level 2: 0xa6
 * Level 3: 0xa5
 * Level 4: 0x9a
 * Level 5: 0x99
 * Level 6: 0x95
 * Level 7: 0x6a
 * Level 8: 0x55

### Command Type: Continuous (0x66)

The arguments are the same as the Nick command.

 * Level 1: 0xa9
 * Level 2: 0xa6
 * Level 3: 0xa5
 * Level 4: 0x9a
 * Level 5: 0x99
 * Level 6: 0x95
 * Level 7: 0x6a
 * Level 8: 0x55
