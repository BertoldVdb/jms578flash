    ; Load settings
    MOV  DPTR, #0x7020
    MOV  A, R0
    MOVX @DPTR, A
    
    INC  DPTR
    MOV  A, R1
    MOVX @DPTR, A
    
    INC  DPTR
    MOV  A, R2
    MOVX @DPTR, A
    
    INC  DPTR
    MOV  A, R3
    MOVX @DPTR, A
    
    INC  DPTR
    MOV  A, R4
    MOVX @DPTR, A
    
    INC  DPTR
    MOV  A, R5
    MOVX @DPTR, A

    MOV  DPTR, #0x7148
    MOV  A, R4
    MOVX @DPTR, A
    
    INC  DPTR
    MOV  A, R5
    MOVX @DPTR, A
    
    MOV  DPTR, #0x7166
    MOV  A, R4
    MOVX @DPTR, A
    
    INC  DPTR
    MOV  A, R5
    MOVX @DPTR, A

    ; Start the transmission
    MOV  DPTR, #0x7142
    MOV  A, #0x12
    MOVX @DPTR, A
    
    MOV  DPTR, #0x714e
    CLR A
    MOVX @DPTR, A
    
    MOV  DPTR, #0x714c
    INC A
    MOVX @DPTR, A
    
    MOV  DPTR, #0x7026
    MOVX @DPTR, A

    RET
