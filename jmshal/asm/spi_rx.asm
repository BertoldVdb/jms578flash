    ; Load transmit buffer
    MOV A, R4
    JZ  noload
load:
    MOVX A, @DPTR
    INC  DPTR
    PUSH DPL
    PUSH DPH

    MOV  DPTR, #0x7140
    MOVX @DPTR, A

    POP DPH
    POP DPL

    DJNZ R4, load
noload:

    ; Load settings
    MOV  DPTR, #0x7160
    MOV  A, R0
    MOVX @DPTR, A
    
    INC  DPTR
    MOV  A, R1
    MOVX @DPTR, A
    
    INC  DPTR
    CLR A
    MOVX @DPTR, A
    
    MOV  DPTR, #0x7148
    MOV  A, R2
    MOVX @DPTR, A
    
    INC  DPTR
    MOV  A, R3
    MOVX @DPTR, A
    
    MOV  DPTR, #0x7166
    MOV  A, R2
    MOVX @DPTR, A
    
    INC  DPTR
    MOV  A, R3
    MOVX @DPTR, A

    ; Start the reception
    MOV  DPTR, #0x7142
    MOV  A, #0x11
    MOVX @DPTR, A
    
    MOV  DPTR, #0x714c
    MOV  A, #1
    MOVX @DPTR, A

    RET
