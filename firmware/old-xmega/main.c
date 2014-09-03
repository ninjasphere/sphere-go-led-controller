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
#include <dma.h>

#include <string.h>

#define F_CPU sysclk_get_cpu_hz()
#include <util/delay.h>

#include "conf_serial.h"
#define DMA_CHANNEL     0

#define PIXEL_WIDTH 16
#define PIXEL_HEIGHT 16
#define PIXEL_SIZE 3
#define FRAME_SIZE (PIXEL_WIDTH * PIXEL_HEIGHT * PIXEL_SIZE)

typedef uint16_t matrix_columns_bits_t; // RR
typedef matrix_columns_bits_t matrix_column_colors_t[PIXEL_SIZE]; // RR GG BB
typedef matrix_column_colors_t matrix_column_pairs_t[2]; // RR GG BB RR GG BB
typedef matrix_column_pairs_t matrix_columns_single_t[8]; // (RR GG BB RR GG BB) * 8, iterate through row each nRow

//volatile uint8_t incomingBuffer[FRAME_SIZE];

typedef struct matrix_frame_
{
	matrix_columns_single_t forBitIndex[8];
} matrix_frame_t;

void board_init(void);

static void dma_init(void);

void matrix_swap_buffers(void);
void matrix_extract_bits(volatile uint8_t *buffer, volatile matrix_frame_t *frame);

static usart_rs232_options_t usart_mainboard_options = {
   .baudrate = USART_MAINBOARD_SERIAL_BAUDRATE,
   .charlength = USART_MAINBOARD_SERIAL_CHAR_LENGTH,
   .paritytype = USART_MAINBOARD_SERIAL_PARITY,
   .stopbits = USART_MAINBOARD_SERIAL_STOP_BIT
};

struct spi_device spi_device_conf = {
	.id = IOPORT_CREATE_PIN(PORTF, 4)
};

volatile matrix_frame_t frames[2];
volatile uint8_t currentFrame = 0;

#define frontFrame (frames[currentFrame])
#define backFrame (frames[1 - currentFrame])

void matrix_rows_off(void);
void matrix_pulse_latch(void);
void matrix_set_row(uint8_t row);
static void screen_row_draw(void);

int main(void)
{
	pmic_init();
	sysclk_init();
	sleepmgr_init();
	board_init();

	// initialise the input UART from the Sphere MainBoard (USARTD0)
    usart_init_rs232(USART_MAINBOARD_SERIAL, &usart_mainboard_options);
    //usart_set_rx_interrupt_level(USART_MAINBOARD_SERIAL, USART_INT_LVL_LO);

    // initialise SPI for communication to the matrix drivers
    ioport_configure_port_pin(&PORTF, PIN4_bm, IOPORT_INIT_HIGH | IOPORT_DIR_OUTPUT); // nCS
    ioport_configure_port_pin(&PORTF, PIN5_bm, IOPORT_INIT_HIGH | IOPORT_DIR_OUTPUT); // MOSI
	ioport_configure_port_pin(&PORTF, PIN7_bm, IOPORT_INIT_HIGH | IOPORT_DIR_OUTPUT); // SCK
	spi_master_init(&SPIF);
	spi_master_setup_device(&SPIF, &spi_device_conf, SPI_MODE_0, 8000000, 0);
	spi_enable(&SPIF);

	// initialise nRow[1..8]
	ioport_configure_group(IOPORT_PORTJ, 0x0f, IOPORT_INIT_LOW | IOPORT_DIR_OUTPUT);
	ioport_configure_group(IOPORT_PORTB, 0xf0, IOPORT_INIT_LOW | IOPORT_DIR_OUTPUT);
	// and latch
	ioport_configure_port_pin(&PORTE, PIN2_bm, IOPORT_INIT_LOW | IOPORT_DIR_OUTPUT);
	// and blank
	ioport_configure_port_pin(&PORTE, PIN3_bm, IOPORT_INIT_HIGH | IOPORT_DIR_OUTPUT);

	// setup the timer/counter for pulsing
	tc_enable(&TCC0);
	tc_set_overflow_interrupt_callback(&TCC0, screen_row_draw);
	tc_set_wgm(&TCC0, TC_WG_NORMAL);
	tc_write_period(&TCC0, 10);
	tc_set_overflow_interrupt_level(&TCC0, TC_INT_LVL_LO);

	dma_init();

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
	/*for ( int x = 0; x < PIXEL_WIDTH; x++ ) {
		for ( int y = 0; y < PIXEL_HEIGHT; y++ ) {
			int i = ((y*PIXEL_WIDTH)+x)*PIXEL_SIZE;

			backFrame.data[i+0] = (x*15); // R
			backFrame.data[i+1] = (y*15); // G
			backFrame.data[i+2] = 0;//0; // B
		}
	}*/
	memset( backFrame.forBitIndex, 1, FRAME_SIZE );

	//matrix_extract_bits( backFrame.data, &backFrame );
	matrix_swap_buffers( );

	matrix_rows_off( );

	usart_putchar(USART_MAINBOARD_SERIAL, 'F');

	// start by sending out a blanking set
	matrix_column_pairs_t columnPairs;
	memset( &columnPairs, 0, 12 );
	spi_select_device(&SPIF, &spi_device_conf);
	spi_write_packet(&SPIF, (const uint8_t *)&columnPairs, 12);
	//spi_deselect_device(&SPIF, &spi_device_conf);

	// ready, go!
	tc_write_clock_source(&TCC0, TC_CLKSEL_DIV1_gc);

	while ( 1 ) {
		sleepmgr_enter_sleep(); // ZZZZzzzzz
	}
}

static struct dma_channel_config dmach_conf;

static void dma_frame_transfer_done(enum dma_channel_status status)
{
	//matrix_extract_bits( backFrame.data, &backFrame );
	matrix_swap_buffers( );
	usart_putchar(USART_MAINBOARD_SERIAL, 'F');

	// reset address and restart
	dma_channel_set_destination_address(&dmach_conf, (uint16_t)(uintptr_t)backFrame.forBitIndex);
	dma_channel_write_config(DMA_CHANNEL, &dmach_conf);
	dma_channel_enable(DMA_CHANNEL);
}

static void dma_init(void)
{
	memset(&dmach_conf, 0, sizeof(dmach_conf));

	// (void *) &USARTD0.DATA,
	dma_channel_set_source_address(&dmach_conf, (uint16_t)(uintptr_t)&(USARTD0.DATA));
	// DMA_CH_SRCRELOAD_NONE_gc, 
	dma_channel_set_src_reload_mode(&dmach_conf, DMA_CH_SRCRELOAD_NONE_gc);
	// DMA_CH_SRCDIR_FIXED_gc,
	dma_channel_set_src_dir_mode(&dmach_conf, DMA_CH_SRCDIR_FIXED_gc);

	// Rx_Buf,
	dma_channel_set_destination_address(&dmach_conf, (uint16_t)(uintptr_t)backFrame.forBitIndex);
	// DMA_CH_DESTRELOAD_NONE_gc, 
	dma_channel_set_dest_reload_mode(&dmach_conf, DMA_CH_DESTRELOAD_NONE_gc);
	// DMA_CH_DESTDIR_INC_gc,
	dma_channel_set_dest_dir_mode(&dmach_conf, DMA_CH_DESTDIR_INC_gc);

	// TEST_CHARS, 
	dma_channel_set_transfer_count(&dmach_conf, FRAME_SIZE);
	// DMA_CH_BURSTLEN_1BYTE_gc, 
	dma_channel_set_burst_length(&dmach_conf, DMA_CH_BURSTLEN_1BYTE_gc);

	dma_channel_set_single_shot(&dmach_conf); // single shot = 1 data xfer instead of 1 block
	//dma_channel_set_repeats(&dmach_conf, 0);
	dma_channel_set_trigger_source(&dmach_conf, DMA_CH_TRIGSRC_USARTD0_RXC_gc);

	dma_enable();

	dma_set_callback(DMA_CHANNEL, dma_frame_transfer_done);
	dma_channel_set_interrupt_level(&dmach_conf, DMA_INT_LVL_LO);

	dma_channel_write_config(DMA_CHANNEL, &dmach_conf);
	dma_channel_enable(DMA_CHANNEL);
}

static uint16_t cnt = 0;
static uint8_t bitIndex = 0;
static uint8_t rowIndex = 0;

static void screen_row_draw(void)
{
	// set rows off
	matrix_rows_off( );



	




	// pulse the latch pin
	matrix_pulse_latch( );
	tc_restart(&TCC0);

	// pulse the row
	matrix_set_row( rowIndex );

	// and prepare the timer
	tc_write_period(&TCC0, (1<<bitIndex) << 7);

	




	// push next column data
	bitIndex = cnt / 8;
	rowIndex = cnt % 8;

	volatile matrix_column_pairs_t *columnPairs = &(frontFrame.forBitIndex[bitIndex][rowIndex]);
	spi_write_packet(&SPIF, (const uint8_t *)columnPairs, 12);

	cnt = (cnt+1) & 0x3f; //% (8*8); // increment with wrap
}

void matrix_swap_buffers( )
{
	currentFrame = (1 - currentFrame);
}

/*
void matrix_extract_bits(volatile uint8_t *buffer, volatile matrix_frame_t *frame)
{
	memset( frame->forBitIndex, 0, FRAME_SIZE );

	// this can probably be optimised and is probably wrong.
	for ( uint8_t nRow = 0; nRow < 8; nRow++ ) {
		for ( uint8_t rowPair = 0; rowPair < 2; rowPair++ ) {
			uint8_t absoluteRow = (rowPair*8) + nRow;

			for ( uint8_t color = 0; color < 3; color++ ) {
				for ( uint8_t column = 0; column < 16; column++ ) {
					uint8_t index = (absoluteRow*PIXEL_WIDTH) + column;
					uint8_t value = buffer[(index*PIXEL_SIZE)+color];

					for ( uint8_t bitIndex = 0; bitIndex < 8; bitIndex++ ) {
						uint16_t pv = ((value & (1<<bitIndex)) != 0) ? 1 : 0;

						frame->forBitIndex[bitIndex][nRow][1-rowPair][color] |= (pv<<((column+8)%16));
					}
				}
			}
		}
	}
}
*/

void matrix_rows_off(void)
{
	ioport_set_pin_high(IOPORT_CREATE_PIN(PORTE,3)); // blank
	ioport_set_group_low(IOPORT_PORTJ, 0x0f);
	ioport_set_group_low(IOPORT_PORTB, 0xf0);
}

void matrix_pulse_latch(void)
{
	ioport_set_pin_high(IOPORT_CREATE_PIN(PORTE,2));
	_delay_us( 10 );
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

/*
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
*/
