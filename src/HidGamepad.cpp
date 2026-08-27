#include "HidGamepad.h"

const uint8_t GAMEPAD_REPORT_DESCRIPTOR[] = {
    0x05, 0x01,        // USAGE_PAGE (Generic Desktop)
    0x09, 0x05,        // USAGE (Game Pad)
    0xA1, 0x01,        // COLLECTION (Application)
    0x85, 0x01,        //   REPORT_ID (1)

    // ---- 16 botões digitais ----
    0x05, 0x09,        //   USAGE_PAGE (Button)
    0x19, 0x01,        //   USAGE_MINIMUM (Button 1)
    0x29, 0x10,        //   USAGE_MAXIMUM (Button 16)
    0x15, 0x00,        //   LOGICAL_MINIMUM (0)
    0x25, 0x01,        //   LOGICAL_MAXIMUM (1)
    0x75, 0x01,        //   REPORT_SIZE (1 bit)
    0x95, 0x10,        //   REPORT_COUNT (16)
    0x81, 0x02,        //   INPUT (Data,Var,Abs)

    // ---- 4 eixos analógicos: X, Y, Z, Rz (8 bits cada) ----
    0x05, 0x01,        //   USAGE_PAGE (Generic Desktop)
    0x09, 0x30,        //   USAGE (X)
    0x09, 0x31,        //   USAGE (Y)
    0x09, 0x32,        //   USAGE (Z)
    0x09, 0x35,        //   USAGE (Rz)
    0x15, 0x00,        //   LOGICAL_MINIMUM (0)
    0x26, 0xFF, 0x00,  //   LOGICAL_MAXIMUM (255)
    0x75, 0x08,        //   REPORT_SIZE (8 bits)
    0x95, 0x04,        //   REPORT_COUNT (4)
    0x81, 0x02,        //   INPUT (Data,Var,Abs)

    0xC0               // END_COLLECTION
};

const uint16_t GAMEPAD_REPORT_DESCRIPTOR_SIZE = sizeof(GAMEPAD_REPORT_DESCRIPTOR);
