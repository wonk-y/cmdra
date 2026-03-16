*** Settings ***
Library    cmdagent_client.robot_library.CmdAgentLibrary    ${ADDRESS}    ${CA_CERT}    ${CLIENT_CERT}    ${CLIENT_KEY}

*** Test Cases ***
Start Argv Command
    ${execution}=    Start Argv    /bin/echo    robot-smoke
    Should Not Be Empty    ${execution.execution_id}
