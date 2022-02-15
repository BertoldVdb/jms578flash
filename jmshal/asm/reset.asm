    CLR EA
    
    ; Disconnect USB
    LCALL 0x2eb7

    ; Reset some configuration
    MOV DPTR, #0x7800
    CLR A
    MOVX @DPTR, A

    ; Reset slow timer
    MOV  DPTR, #0x7078
    MOVX @DPTR, A

wait:
    MOVX A, @DPTR
    CJNE A, #10, wait

    MOV  DPTR, #0x7004
    MOV  A, #0x57
    MOVX @DPTR, A

loop:
    SJMP loop
