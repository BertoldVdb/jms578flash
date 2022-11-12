.equ SCSI_IN,  0x7990
.equ SCSI_OUT, 0x3500
.equ TMP,      0x20

hookCheck:
    MOV   DPTR, #SCSI_IN 
    MOVX  A, @DPTR
    CJNE  A, #0x77, hookNext

    PUSH  TMP
    LCALL hookRun
    POP   TMP

    ; Read back ACC
    MOV   DPTR, #SCSI_OUT
    MOVX  @DPTR, A

    ; Read back registers
    MOV   A, R0
    INC   DPTR
    MOVX  @DPTR, A
    MOV   A, R1
    INC   DPTR
    MOVX  @DPTR, A
    MOV   A, R2
    INC   DPTR
    MOVX  @DPTR, A
    MOV   A, R3
    INC   DPTR
    MOVX  @DPTR, A
    MOV   A, R4
    INC   DPTR
    MOVX  @DPTR, A
    MOV   A, R5
    INC   DPTR
    MOVX  @DPTR, A
    MOV   A, R6
    INC   DPTR
    MOVX  @DPTR, A
    MOV   A, R7
    INC   DPTR
    MOVX  @DPTR, A

hookRespond:
    ; Send simple read response back
    MOV 0x57, #2
    MOV 0x59, #0
    MOV 0x5a, #0x9

    RET

hookNext:
    CJNE  A, #0x78, hookOrig
    MOV   DPTR, #SCSI_OUT
    MOV   A, #0xAA
    MOVX  @DPTR, A
    INC   DPTR
    MOV   A, #0xBB
    MOVX  @DPTR, A
    SJMP hookRespond
   
hookOrig:
    LJMP  0xdead ; Original handler

hookRun:
    ; Copy 4 bytes to stack
    MOV R0, #4
loop:
    INC   DPTR
    MOVX  A, @DPTR
    MOV   TMP, A
    PUSH  TMP
    DJNZ  R0, loop
    
    ; Load all registers
    INC   DPTR
    MOVX  A, @DPTR
    MOV   R0, A
    INC   DPTR
    MOVX  A, @DPTR
    MOV   R1, A
    INC   DPTR
    MOVX  A, @DPTR
    MOV   R2, A
    INC   DPTR
    MOVX  A, @DPTR
    MOV   R3, A
    INC   DPTR
    MOVX  A, @DPTR
    MOV   R4, A
    INC   DPTR
    MOVX  A, @DPTR
    MOV   R5, A
    INC   DPTR
    MOVX  A, @DPTR
    MOV   R6, A
    INC   DPTR
    MOVX  A, @DPTR
    MOV   R7, A
    
    ; Load ACC
    INC   DPTR
    MOVX  A, @DPTR
  
    ; Set DPTR
    POP DPH
    POP DPL

    ; Jump to handler function
    RET

