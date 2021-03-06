# <h.zeller@acm.org>
##

# CPU frequency and baudrate as used at compile time. We want to have
# a slow communication as we have pretty long lines and don't use RS422, so
# consider to reduce the baudrate for production use to something that works on
# the line (looks like 9600 is pretty ok for 10meters)
#
# The FEATURE_* flags allow to switch features by setting them to 0 or 1
#
# FEATURE_RFID_DEBUG provides query feature for RC522 status
#    (+= ~298 bytes)
#
# FEATURE_BAUD_CHANGE allows to change baudrate via command. Good
#    for debugging and speed-testing. But uses a quite a few bytes
#    space in firmware, and fixed speed might be more useful in the field.
#    (+= ~472 bytes)
#
# SERIAL_BAUDRATE is the baudrate used by the terminal. If FEATURE_BAUD_CHANGE
#    is switched on, then this is the initial default baudrate.
#
# FEATURE_LCD switches between LCD and RGB LED support =1: LCD, =0: LED
#    (LCD: += ~264 bytes)
#
# FIXED_TERMINAL_NAME : if defined (needs to be a string), then the name of the
# terminal is compiled in and cannot be changed via EEPROM (-= ~420 bytes, so
# saves code memory).
#
GIT_VERSION="$(shell git log --date=short --pretty=format:'[%h] @ %cd' -n 1)"

DEFINES=-DGIT_VERSION='$(GIT_VERSION)' -DF_CPU=8000000UL -DFEATURE_RFID_DEBUG=0 \
        -DFEATURE_BAUD_CHANGE=0 -DSERIAL_BAUDRATE=9600 \
        -DFEATURE_LCD=0 # -DFIXED_TERMINAL_NAME='"gate"'

TARGET_ARCH=-mmcu=atmega8
CXX=avr-g++
CXXFLAGS=-Os -g -Wall -ffunction-sections -fdata-sections -mcall-prologues $(DEFINES)
AVRDUDE_DEVICE ?= /dev/ttyUSB0
AVRDUDE     = avrdude -p m8 -c stk500v2 -P $(AVRDUDE_DEVICE)
FLASH_CMD   = $(AVRDUDE) -e -U flash:w:main.hex
LINK=avr-g++ -g $(TARGET_ARCH) -Wl,-gc-sections
OBJECTS=terminal-main.o lcd.o serial-com.o keypad.o tone-gen.o mfrc522/mfrc522.o mfrc522-debug.o

all : main.hex

main.elf: $(OBJECTS)
	$(LINK) -o $@ $(OBJECTS)
	avr-size $@

# To compare different sized assembly outputs. Creates files with names including
# size
space-output: main.elf
	avr-objdump -S $^ > /tmp/terminal-`avr-size -d main.elf | grep main.elf | awk '{print $$4}'`.S

disasm: main.elf
	avr-objdump -C -S main.elf

main.hex: main.elf
	avr-objcopy -j .text -j .data -O ihex main.elf main.hex

eeprom.hex: main.elf
	avr-objcopy -j .eeprom --change-section-lma .eeprom=0 -O ihex main.elf $@

read-eeprom:
	$(AVRDUDE) -qq -U eeprom:r:-:r | od -t x1zu2

flash: main.hex
	$(FLASH_CMD)

eeprom-flash: eeprom.hex
	$(AVRDUDE) -U eeprom:w:eeprom.hex

clean:
	rm -f $(OBJECTS) main.elf main.hex

# Documentation page references from
# http://www.atmel.com/images/atmel-2486-8-bit-avr-microcontroller-atmega8_l_datasheet.pdf / pp 216-217 are interesting.
# internal oscillator.
### Fuse high byte:
# 7 rstdisbl	1   disable external reset: disabled (i.e.: reset enabled).
# 6 wdton	1   watchdog.
# - spien	0   serial programming: enabled.
# 4 chkopt	1   clock options (doesn't matter, we do internal RC)
#
# 3 eesave      0   save eeprom on chip erase: enabled
# 2 bootsz1	1
# 1 bootsz0	1
# 0 bootrst     1   starting at 0x0000

### low byte:
# 7 bodlevel	1   brown out detect at 2.7v
# 6 boden	0   brown out enable.
# 5 sut1	1-+ slow rising power (page 30)
# 4 sut0	0/
#
# 3 cksel3	0\
# 2 cksel2	1 + internal RC oscillator  (page 30)  (8Mhz)
# 1 cksel1	0/
# 0 cksel0	0
fuse:
	$(AVRDUDE) -U hfuse:w:0xd7:m -U lfuse:w:0xe4:m
