#include <avr/io.h>
#include <stdint.h>

static inline void SPI_write(uint8_t byte) {
	SPDR = byte;
	while ( (SPSR & (1<<SPIF)) == 0){}
}

static inline void SPI_init(){
    DDRB |= (1<<3);
    DDRB |= (1<<5);

    SPCR = (0<<SPIE) | //We don't want interrupts
    (1<<SPE) | //We do want the SPI enabled
    (0<<DORD) | //We want the data to be shifted out LSB
    (1<<MSTR) | //We want the atmega to be a master
    (0<<CPOL) | //We want the leading edge to be rising
    (0<<CPHA) | //We want the leading edge to be sample
    (0<<SPR1) | (0<<SPR0) ; // sets the clock speed

    SPSR = (0<<SPIF) | // SPI interrupt flag
    (0<<WCOL) | //Write collision flag
    (1<<SPI2X) ; //Doubles the speed of the SPI clock
}
