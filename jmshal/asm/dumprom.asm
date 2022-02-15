    ; Do not call other addresses in this firmware
    CLR  0x20.1

    ; Do not map CODE to 0x8000
    MOV  DPTR, #0x708c
    MOV  A, #7
    MOVX @DPTR, A

    CLR  A

    ; Do not provide firmware version over USB
    ; This confuses the windows flash tool, since
    ; this firmware has no flashing capability.

    MOV  DPTR, #0x411a
    MOVX @DPTR, A
    INC  DPTR
    MOVX @DPTR, A
    INC  DPTR
    MOVX @DPTR, A
    INC  DPTR
    MOVX @DPTR, A
    INC  DPTR

    MOV  R3, A

    ; Copy from 0x00
    MOV  R7, A
    MOV  R6, A

    ; to 0x800
    MOV  R5, A
    MOV  R4, #0x80

loop:
    LCALL 0x1f1b ;memcpy from code to xdata
    INC   R4
    INC   R6
    CJNE  R6, #0x40, loop

    ; Turn on LED to indicate the FW is running
    MOV   DPTR, #0x7054
    MOV   A, #0xe1
    MOVX  @DPTR, A

    RET
