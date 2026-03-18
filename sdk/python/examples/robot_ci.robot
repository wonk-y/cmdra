*** Settings ***
Library    cmdra_client.robot_library.CmdraLibrary    ${ADDRESS}    ${CA_CERT}    ${CLIENT_CERT}    ${CLIENT_KEY}

*** Test Cases ***
List Executions Works
    ${items}=    List Executions
    Should Not Be Equal    ${items}    ${None}

Write Stdin Works
    ${execution}=    Start Shell Command    read line; printf '%s\n' "$line"    shell_binary=/bin/sh
    ${stdin_line}=    Catenate    SEPARATOR=    robot-write-stdin    ${\n}
    Write Stdin    ${execution.execution_id}    ${stdin_line}    eof=${True}
    Sleep    0.2s
    ${details}=    Get Execution With Output    ${execution.execution_id}
    ${contains}=    Evaluate    any(b"robot-write-stdin" in chunk.data for chunk in $details.output if not chunk.eof)
    Should Be True    ${contains}
