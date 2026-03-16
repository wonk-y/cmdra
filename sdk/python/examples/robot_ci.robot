*** Settings ***
Library    cmdra_client.robot_library.CmdraLibrary    ${ADDRESS}    ${CA_CERT}    ${CLIENT_CERT}    ${CLIENT_KEY}

*** Test Cases ***
List Executions Works
    ${items}=    List Executions
    Should Not Be Equal    ${items}    ${None}
