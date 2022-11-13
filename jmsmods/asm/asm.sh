#!/bin/sh

as31 -Fbin hook.asm
as31 -Fbin reset.asm
as31 -Fbin spi_tx.asm
as31 -Fbin spi_rx.asm
as31 -Fbin disable.asm
