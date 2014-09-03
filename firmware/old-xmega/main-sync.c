/*

Matrix is wired up as 2 half matrix modules (8x16), sharing row selects.

Serial data is shifted out through shift registers, functionally identical to:

MOSI -> B1 -> G1 -> R1 -> B2 -> G2 -> R2 -> (null)

Where each shift register ([R,G,B][1,2]) is 16 bits, representing each column.

Data is pushed through the shift registers in the reverse order,
so the bytes to send (in order) are: RR GG BB RR GG BB

The sequence for a screen refresh is:
1) Push column data (12 bytes) RR GG BB RR GG BB
2) Set all nRow[1..8] = HI (off)
3) Pulse the latch pin
4) Iterate through nRow[1..8] (repeating above steps) pulling one low and delaying for a PWM period

The above is then repeated (with the column data pushing in parallel)
8 times for the each bit of the colour space, with the PWM period proportional
to the significance of the bit.

For example, bit (1<<0) might be held for 1ms while (1<<7) might be held for 128ms.

*/


#include <board.h>
#include <compiler.h>
#include <gpio.h>
#include <nvm.h>
#include <sysclk.h>
#include <tc.h>
#include <usart.h>
#include <spi_master.h>

#include <string.h>

#define F_CPU sysclk_get_cpu_hz()
#include <util/delay.h>

#include "conf_serial.h"

#define PIXEL_WIDTH 16
#define PIXEL_HEIGHT 16
#define PIXEL_SIZE 3
#define FRAME_SIZE (PIXEL_WIDTH * PIXEL_HEIGHT * PIXEL_SIZE)

typedef uint16_t matrix_columns_bits_t; // RR
typedef matrix_columns_bits_t matrix_column_colors_t[PIXEL_SIZE]; // RR GG BB
typedef matrix_column_colors_t matrix_column_pairs_t[2]; // RR GG BB RR GG BB
typedef matrix_column_pairs_t matrix_columns_single_t[8]; // (RR GG BB RR GG BB) * 8, iterate through row each nRow

typedef struct matrix_frame_
{
	uint8_t data[FRAME_SIZE];

	matrix_columns_single_t forBitIndex[8];
} matrix_frame_t;

void board_init(void);
void matrix_swap_buffers(void);
void matrix_extract_bits(matrix_frame_t *frame);

static usart_rs232_options_t usart_mainboard_options = {
   .baudrate = USART_MAINBOARD_SERIAL_BAUDRATE,
   .charlength = USART_MAINBOARD_SERIAL_CHAR_LENGTH,
   .paritytype = USART_MAINBOARD_SERIAL_PARITY,
   .stopbits = USART_MAINBOARD_SERIAL_STOP_BIT
};

struct spi_device spi_device_conf = {
	.id = IOPORT_CREATE_PIN(PORTF, 4)
};

matrix_frame_t frames[2];
volatile uint8_t currentFrame = 0;

#define frontFrame (frames[currentFrame])
#define backFrame (frames[1 - currentFrame])

void matrix_rows_off(void);
void matrix_pulse_latch(void);
void matrix_set_row(uint8_t row);

int main(void)
{
	sysclk_init();
	board_init();

	// initialise the input UART from the Sphere MainBoard (USARTD0)
    usart_init_rs232(USART_MAINBOARD_SERIAL, &usart_mainboard_options);
    usart_set_rx_interrupt_level(USART_MAINBOARD_SERIAL, USART_INT_LVL_LO);

    // initialise SPI for communication to the matrix drivers
    ioport_configure_port_pin(&PORTF, PIN4_bm, IOPORT_INIT_HIGH | IOPORT_DIR_OUTPUT); // nCS
    ioport_configure_port_pin(&PORTF, PIN5_bm, IOPORT_INIT_HIGH | IOPORT_DIR_OUTPUT); // MOSI
	ioport_configure_port_pin(&PORTF, PIN7_bm, IOPORT_INIT_HIGH | IOPORT_DIR_OUTPUT); // SCK
	spi_master_init(&SPIF);
	spi_master_setup_device(&SPIF, &spi_device_conf, SPI_MODE_0, 1000000, 0);
	spi_enable(&SPIF);

	// initialise nRow[1..8]
	ioport_configure_group(IOPORT_PORTJ, 0x0f, IOPORT_INIT_LOW | IOPORT_DIR_OUTPUT);
	ioport_configure_group(IOPORT_PORTB, 0xf0, IOPORT_INIT_LOW | IOPORT_DIR_OUTPUT);
	// and latch
	ioport_configure_port_pin(&PORTE, PIN2_bm, IOPORT_INIT_LOW | IOPORT_DIR_OUTPUT);
	// and blank
	ioport_configure_port_pin(&PORTE, PIN3_bm, IOPORT_INIT_HIGH | IOPORT_DIR_OUTPUT);

	pmic_enable_level(PMIC_LVL_LOW);
	cpu_irq_enable();

	_delay_ms( 255 );
	_delay_ms( 255 );
	_delay_ms( 255 );
	_delay_ms( 255 );
	_delay_ms( 255 );
	_delay_ms( 255 );
	_delay_ms( 255 );
	_delay_ms( 255 );
	_delay_ms( 255 );
	_delay_ms( 255 );
//#define SAFE
#ifdef SAFE
	spi_select_device(&SPIF, &spi_device_conf);
	uint8_t blanks[32];
	memset( blanks, 0x00, 32 );
	blanks[31] = 0x00; // B
	blanks[30] = 0x10; // B

	blanks[29] = 0x00; // G
	blanks[28] = 0x10; // G

	blanks[27] = 0x00; // R
	blanks[26] = 0x10; // R

	blanks[25] = 0x10; // B
	blanks[24] = 0x00; // B

	blanks[23] = 0x10; // G
	blanks[22] = 0x00; // G

	blanks[21] = 0x10; // R
	blanks[20] = 0x00; // R

	while ( 1 ) {
		spi_write_packet( &SPIF, blanks, 32 );
		matrix_pulse_latch( );
		_delay_ms( 1 );
		ioport_set_pin_low(IOPORT_CREATE_PIN(PORTE,3));

		// all rows
		ioport_set_group_high(IOPORT_PORTJ, 0x01);
		//ioport_set_group_high(IOPORT_PORTB, 0xf0);
	}
	spi_deselect_device(&SPIF, &spi_device_conf);
#endif

	// dummy data
	for ( int x = 0; x < PIXEL_WIDTH; x++ ) {
		for ( int y = 0; y < PIXEL_HEIGHT; y++ ) {
			int i = ((y*PIXEL_WIDTH)+x)*PIXEL_SIZE;
			backFrame.data[i+0] = 0; // R
			backFrame.data[i+1] = (x*15); // G
			backFrame.data[i+2] = (y*15); // B
		}
	}

	matrix_extract_bits( &backFrame );
	matrix_swap_buffers( );


	usart_putchar(USART_MAINBOARD_SERIAL, 'F');

	// FIXME: refactor to interrupt based so we can sleep
	while ( 1 ) {
		uint8_t bitIndex = 0, rowIndex = 0;

		for ( bitIndex = 0; bitIndex < 8; bitIndex++ ) {
			//matrix_columns_single_t *bitFrame = &(frontFrame.forBitIndex[bitIndex]);

			for ( rowIndex = 0; rowIndex < 8; rowIndex++ ) {
				matrix_column_pairs_t *columnPairs = &(frontFrame.forBitIndex[bitIndex][rowIndex]);

				// push column data
				spi_select_device(&SPIF, &spi_device_conf);
				spi_write_packet(&SPIF, (const uint8_t *)columnPairs, 12);
				spi_deselect_device(&SPIF, &spi_device_conf);

				// set rows off
				matrix_rows_off( );

				// pulse the latch pin
				matrix_pulse_latch( );

				// pulse the row
				matrix_set_row( rowIndex );

				for ( int i = 0; i < (1<<bitIndex); i++ ) {
					// sum of 1+2+4+8...+128 = 256
					// so we need 256 of this delay to run in 1 second / 30 frames / 8 rows > 4ms = 4000us
					_delay_us( 1 );
				}

				// and off again
				matrix_rows_off( );
			}
		}
	}
}

void matrix_swap_buffers( )
{
	currentFrame = (1 - currentFrame);
}

/*
typedef uint16_t matrix_columns_bits_t; // RR
typedef matrix_columns_bits_t matrix_column_colors_t[PIXEL_SIZE]; // RR GG BB
typedef matrix_column_colors_t matrix_column_pairs_t[2]; // RR GG BB RR GG BB
typedef matrix_column_pairs_t matrix_columns_single_t[8]; // (RR GG BB RR GG BB) * 8, iterate through row each nRow

forBitIndex[0..7]

forBitIndex[BITVALUE][nROW][top,bottom][r,g,b]
*/

void matrix_extract_bits(matrix_frame_t *frame)
{
	memset( frame->forBitIndex, 0, FRAME_SIZE );

	// this can probably be optimised and is probably wrong.
	for ( uint8_t nRow = 0; nRow < 8; nRow++ ) {
		for ( uint8_t rowPair = 0; rowPair < 2; rowPair++ ) {
			uint8_t absoluteRow = (rowPair*8) + nRow;

			for ( uint8_t color = 0; color < 3; color++ ) {
				for ( uint8_t column = 0; column < 16; column++ ) {
					uint8_t index = (absoluteRow*PIXEL_WIDTH) + column;
					uint8_t value = frame->data[(index*PIXEL_SIZE)+color];

					for ( uint8_t bitIndex = 0; bitIndex < 8; bitIndex++ ) {
						uint16_t pv = (value & (1<<bitIndex)) ? 1 : 0;

						frame->forBitIndex[bitIndex][nRow][1-rowPair][color] |= (pv<<((column+8)%16));
					}
				}
			}
		}
	}
}

void matrix_rows_off(void)
{
	ioport_set_pin_high(IOPORT_CREATE_PIN(PORTE,3)); // blank
	ioport_set_group_low(IOPORT_PORTJ, 0x0f);
	ioport_set_group_low(IOPORT_PORTB, 0xf0);
}

void matrix_pulse_latch(void)
{
	ioport_set_pin_high(IOPORT_CREATE_PIN(PORTE,2));
	_delay_us( 100 );
	ioport_set_pin_low(IOPORT_CREATE_PIN(PORTE,2));
}

void matrix_set_row(uint8_t row)
{
	uint8_t val = (1<<row);
	ioport_set_group_high(IOPORT_PORTJ, val & 0x0f);
	ioport_set_group_high(IOPORT_PORTB, val & 0xf0);
	ioport_set_pin_low(IOPORT_CREATE_PIN(PORTE,3)); // blank
}

void board_init()
{

}

static uint16_t currentIndex = 0;

ISR(USART_MAINBOARD_SERIAL_RX_Vect)
{
    if (usart_rx_is_complete(USART_MAINBOARD_SERIAL))
	{
		uint8_t data;
		data = USARTD0.DATA;

		usart_putchar(USART_MAINBOARD_SERIAL, 'P');
		backFrame.data[currentIndex] = data;
		currentIndex++;

		// check for frame completion
		if (currentIndex == FRAME_SIZE)
		{
			matrix_extract_bits( &backFrame );
			matrix_swap_buffers( );
			currentIndex = 0;
			usart_putchar(USART_MAINBOARD_SERIAL, 'F');
		}
	}
}
