#ifndef F_CPU
#define F_CPU 11059200
#endif

#include <avr/io.h>
#include <avr/pgmspace.h>
#include <avr/interrupt.h>
#include <util/delay.h>
#include <string.h>
#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <stdbool.h>

#define VERSION "1.0.1"

#define OSD_DEBUG 0

#define BAUD_RATE 230400
#define BAUD_UBRR (((F_CPU / (BAUD_RATE * 16UL))) - 1)

#define CMD_NOP 0
#define CMD_WRITE_BUFFER 1
#define CMD_SWAP_BUFFERS 2
#define CMD_READ_RAW_TEMP 'R'
#define CMD_READ_TEMP     'T'
#define CMD_READ_VER      'V'

#define ROW_BYTES (16*3*2)
#define ROWS 8
#define BUFFER_BYTES (ROWS * ROW_BYTES)

uint16_t raw_adc_T;
uint16_t adc_T;
uint16_t adc_T_old;
uint8_t  dimmer;

#define FADE_STEPS 5
uint8_t  fade_in[FADE_STEPS]           = { 0xff, 0xb4, 0x80, 0x5a, 0x40};
uint16_t adc_threshold_in[FADE_STEPS]  = {0x1e5,0x1df,0x1d1,0x1c4,0x1b7};
uint8_t  fade_out[FADE_STEPS]          = { 0xff, 0xb4, 0x80, 0x5a, 0x40};
uint16_t adc_threshold_out[FADE_STEPS] = {0x3ff,0x1d1,0x1c4,0x1b7,0x1aa};


uint8_t buffer1[BUFFER_BYTES];
uint8_t buffer2[BUFFER_BYTES];
uint8_t* buffers[2] = {
	buffer1,
	buffer2
};
int frontBuffer = 0;
uint8_t *backBufferPtr = NULL;
uint8_t *frontBufferPtr = NULL;
int backBufferIdx = 0;
bool loaderMode = true;
bool swapFlag   = false;

//uint8_t row[96] = {0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x04, 0x04, 0x04, 0x09, 0x09, 0x09, 0x10, 0x10, 0x10, 0x19, 0x19, 0x19, 0x24, 0x24, 0x24, 0x31, 0x31, 0x31, 0x40, 0x40, 0x40, 0x51, 0x51, 0x51, 0x64, 0x64, 0x64, 0x79, 0x79, 0x79, 0x90, 0x90, 0x90, 0xa9, 0xa9, 0xa9, 0xc4, 0xc4, 0xc4, 0xe1, 0xe1, 0xe1, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x04, 0x04, 0x04, 0x09, 0x09, 0x09, 0x10, 0x10, 0x10, 0x19, 0x19, 0x19, 0x24, 0x24, 0x24, 0x31, 0x31, 0x31, 0x40, 0x40, 0x40, 0x51, 0x51, 0x51, 0x64, 0x64, 0x64, 0x79, 0x79, 0x79, 0x90, 0x90, 0x90, 0xa9, 0xa9, 0xa9, 0xc4, 0xc4, 0xc4, 0xe1, 0xe1, 0xe1};

char state = CMD_NOP;

static inline void UART_init() {
	UBRR0H = (uint8_t)(BAUD_UBRR>>8);
	UBRR0L = (uint8_t)(BAUD_UBRR);
	UCSR0B = (1<<RXEN0)|(1<<TXEN0)|(1<<RXCIE0);
	UCSR0C = ((1<<UCSZ00)|(1<<UCSZ01));
}

void USART_send(unsigned char data){
	while(!(UCSR0A & (1<<UDRE0)));
	UDR0 = data;
}

void USART_send_str(char *s){
	while ( *s != 0)
	  USART_send(*s++);
}

ISR(USART_RX_vect) {
	char cmd = UDR0;
	char temp[12];

	if (state == CMD_WRITE_BUFFER) {
		backBufferPtr[backBufferIdx] = cmd;
		backBufferIdx++;

		if (backBufferIdx == BUFFER_BYTES) {
			// done a whole frame
			state = CMD_NOP;
		}
	} else if (cmd == CMD_SWAP_BUFFERS) {
		loaderMode = false;
		swapFlag = true;
	} else if (cmd == CMD_WRITE_BUFFER) {
		backBufferIdx = 0;
		state = CMD_WRITE_BUFFER;
	} else if (cmd == CMD_READ_RAW_TEMP) {
		sprintf(temp,"0x%03x\r\n",raw_adc_T);
		USART_send_str(temp);
	} else if (cmd == CMD_READ_TEMP) {
		sprintf(temp,"0x%03x\r\n",adc_T);
		USART_send_str(temp);
	} else if (cmd == CMD_READ_VER) {
                strcpy(temp,"V");
                strcat(temp,VERSION);
                strcat(temp,"\r\n");
		USART_send_str(temp);
        }

}

static inline void swap_buffer() {
        frontBuffer = !frontBuffer;
        frontBufferPtr = buffers[frontBuffer];
        backBufferPtr = buffers[!frontBuffer];

}

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


inline void blank_on() {
	PORTD &= ~8;
}

inline void blank_off() {
	PORTD |= 8;
}

inline void latch_on() {
	PORTB |= 0x02;
}

inline void latch_off() {
	PORTB &= ~0x02;
}

inline void toggle_latch() {
	latch_on();
	blank_off();
	latch_off();
}

inline void clrRow() {
	PORTD &= ~0xf0;
	PORTC &= ~0x0f;
}

void setRow(unsigned char rowNum) {
	unsigned char row = 1 << rowNum;
//	clrRow();
	PORTD |= row & 0xf0;
	PORTC |= row & 0x0f;
}

void TLC5951_init(uint8_t dim) {

        latch_on();

        for ( int j = 0; j < 4; j++ ) {

                for ( int i = 0; i < 9; i++ ) { // 288-216
                        SPI_write(0);
                }

                SPI_write(0); // 215-208 bits
                SPI_write(0); // 207-200 bits

//              SPI_write(0x1f); // 199-192 bits
                SPI_write(0x7f); // 199-192 bits

                for ( int i = 0; i < 3; i++ ) { // 191-168 bits
                        SPI_write(dim);
                }
                for ( int i = 0; i < 21; i++ ) { // 168 bits
                        SPI_write(0xff);
                }

        }

        latch_off();
        _delay_ms(1);
        latch_on();
        _delay_ms(1);
        latch_off();

}


void debug_bits(uint16_t x, uint8_t n, uint8_t *row) {
#if OSD_DEBUG
    for (int j = 0; j < n ; j++) {
        uint8_t p;
        if ((x>>j)&0x01)
            p = 255;
        else
            p = 0;
        for ( int k=0 ; k<3 ; k++ )
            row[j*3+k]=p;
   }
#endif
}

static inline void drawPixel(uint16_t x, uint16_t y, uint8_t r, uint8_t g, uint8_t b) {
	uint16_t row = y % 8;
	uint16_t mul = y / 8;
	uint16_t ofs = ROW_BYTES - ((x+1) * 3) - (mul * 16 * 3);

	uint16_t idx = (row * ROW_BYTES) + ofs;

	frontBufferPtr[idx+0] = b;
	frontBufferPtr[idx+1] = g;
	frontBufferPtr[idx+2] = r;
}

static inline void updatePos(int *x, int *y, int *dx, int *dy) {
	*x += *dx;
	*y += *dy;

	if (*dx == 1 && *x == 15) {
		*dx = 0;
		*dy = 1;
	} else if (*dy == 1 && *y == 15) {
		*dx = -1;
		*dy = 0;
	} else if (*dx == -1 && *x == 0) {
		*dx = 0;
		*dy = -1;
	} else if (*dy == -1 && *y == 0) {
		*dx = 1;
		*dy = 0;
	}
}

#define TAIL_LENGTH 8

static void drawSingleLoader(int *x, int *y, int *dx, int *dy) {
	// firstly, update the start pointer
	updatePos( x, y, dx, dy );

	// now, draw our pixels, incrementing the tail length ahead
	{
		int tmp_x = *x;
		int tmp_y = *y;
		int tmp_dx = *dx;
		int tmp_dy = *dy;

		drawPixel(tmp_x,tmp_y,0,0,0);

		for (uint16_t i = 0; i < TAIL_LENGTH; i++) {
			uint8_t col = ((uint16_t)(2 << i))-1;
			updatePos(&tmp_x, &tmp_y, &tmp_dx, &tmp_dy);
			drawPixel(tmp_x,tmp_y,col,col,col);
		}
	}
}

static void drawLoader() {
	static int x1 = 8;
	static int y1 = 0;
	static int dx1 = 1;
	static int dy1 = 0;

	static int x2 = 8;
	static int y2 = 15;
	static int dx2 = -1;
	static int dy2 = 0;

	drawSingleLoader( &x1, &y1, &dx1, &dy1 );
	drawSingleLoader( &x2, &y2, &dx2, &dy2 );
}

uint16_t filter_T(uint16_t newT) {
#define T_n 4
//  static uint16_t T_hist[T_n] = {0,0,0,0,0,0,0,0};
  static uint16_t T_hist[T_n] = {0,0,0,0};
  static uint16_t T_sum = 0;
  static uint8_t  T_cnt = 0;

  T_sum += newT;
  T_sum -= T_hist[T_cnt];
  T_hist[T_cnt] = newT;
  T_cnt++;
  T_cnt %= T_n;

  return T_sum/T_n;
}

int main() {
	for (int i = 0; i < 5; i++) {
		_delay_ms(200);
	}

	// nRow
//	setRow(0);
	clrRow();
	DDRD |= 0xf0;
	DDRC |= 0x0f;

	// latch
	PORTB &= ~0x02;
	DDRB  |= 0x02;

	// blank
	blank_on();
	DDRD |= 8;

	// DCSCLK forced stable low
	PORTC &= (1<<4);
	DDRC  |= (1<<4);



	memset( buffers[0], 0, BUFFER_BYTES );
	memset( buffers[1], 0, BUFFER_BYTES );
	frontBufferPtr = buffers[frontBuffer];
	backBufferPtr = buffers[!frontBuffer];

	UART_init();

	USART_send_str("LED\r\n");

	sei();

	SPI_init();

	dimmer = 0xff;
	TLC5951_init(dimmer);


        // ADC config
        ADMUX = 0x07;
//      ADMUX = (1<<ADLAR) + 0x0e;
	ADCSRA |= (1<<ADSC)|(1<<ADEN);
	adc_T = 0;
	adc_T_old = 0;


	uint16_t i = 0;
	while(1) {
		uint8_t *row = &frontBufferPtr[(i % 8)*ROW_BYTES];

		if (loaderMode && i%(8*8) == 0) {
			drawLoader();
		}

		for ( int outIndex = 0; outIndex < (36 * 4)/3; outIndex++ ) {
			uint8_t pixelIndex = outIndex*2;
			uint8_t pixelA = row[pixelIndex];
			uint8_t pixelB = row[pixelIndex+1];
			SPI_write(pixelA >> 4);
			SPI_write(pixelA << 4);
			SPI_write(pixelB);
		}

		blank_on();
		clrRow();
		_delay_us(3);
		setRow(i % 8);
		toggle_latch();
//		blank_off();

		i++;

		if ( swapFlag && (i%8)==0 ) {
			swap_buffer();
			swapFlag = false;
			USART_send('F');
		}

		if (((ADCSRA & (1<<ADSC))==0)&&((i%8)==0)) {
			raw_adc_T = ADCL | (ADCH<<8);
			ADCSRA |= (1<<ADSC);
			adc_T = filter_T(raw_adc_T);
			debug_bits(adc_T,10,row);

			if (adc_T > adc_T_old) { // cooling
			  for (int k = 0 ; k<FADE_STEPS ; k++)
			    if (adc_T > adc_threshold_in[k]) {
			      if (dimmer < fade_in[k]) {
			        blank_on();
			        dimmer = fade_in[k];
			        TLC5951_init(dimmer);
			        blank_off();
			        break;
			      }
			    }
			}
			else if (adc_T < adc_T_old) { // heating
			  for (int k = FADE_STEPS-1 ; k>=0 ; k--)
			    if (adc_T < adc_threshold_out[k]) {
			      if (dimmer > fade_out[k]) {
			        blank_on();
			        dimmer = fade_out[k];
			        TLC5951_init(dimmer);
			        blank_off();
			        break;
			      }
			    }
			}
			adc_T_old = adc_T;
		}
		if ((i%8)==6)
			debug_bits(dimmer,8,row);
		if ((i%8)==7) {
			uint8_t t;
			t = (uint8_t) 199.1700229-0.3031661571*adc_T; //ADC to temp
			t = (t/10)*16 + t%10; // to BCD
			debug_bits(t,8,row);

		}

	}

	return 0;
}
