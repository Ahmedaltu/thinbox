*** Settings ***
Documentation    Reusable keywords for thinbox system tests.
Library          Process
Library          OperatingSystem

*** Variables ***
${BINARY}        ${CURDIR}/../thinbox
${IMAGE}         alpine
${TIMEOUT}       30s

*** Keywords ***

Build Thinbox Binary
    [Documentation]    Compile the thinbox binary before the suite runs.
    ...                Skips if a pre-built binary is already present.
    ${exists}=    Run Keyword And Return Status    File Should Exist    ${BINARY}
    IF    not ${exists}
        Log    Building thinbox from source...    console=True
        ${result}=    Run Process    go    build    -o    thinbox    .
        ...           cwd=${CURDIR}/..    timeout=60s    stderr=STDOUT
        Should Be Equal As Integers    ${result.rc}    0
        ...    msg=go build failed:\n${result.stdout}
        Log    Build successful.    console=True
    ELSE
        Log    Pre-built binary found, skipping build.    console=True
    END

Run Thinbox Command
    [Documentation]    Run a command inside a thinbox container.
    ...
    ...                Usage: Run Thinbox Command    <cmd>    [arg1]    [arg2] ...
    ...                Translates to: thinbox run alpine <cmd> [args]
    ...
    ...                Returns the Process result object (rc, stdout, stderr).
    [Arguments]    ${cmd}    @{args}
    ${result}=    Run Process
    ...           ${BINARY}    run    ${IMAGE}    ${cmd}    @{args}
    ...           timeout=${TIMEOUT}    stderr=STDOUT
    RETURN    ${result}

Cleanup Test Artifacts
    [Documentation]    Remove any temporary files created during tests.
    Run Keyword And Ignore Error    Remove File    /tmp/thinbox_test_*
