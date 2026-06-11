*** Settings ***
Documentation       System-level tests for thinbox container runtime.
...                 Tests tagged 'requires-linux' need a real Linux kernel
...                 (not WSL2) — they run in CI but are skipped locally.
...
...                 Local (WSL2):   robot --exclude requires-linux tests/
...                 CI (Ubuntu VM): robot tests/
Library             Process
Library             OperatingSystem
Resource            ../resources/thinbox_keywords.resource
Suite Setup         Build Thinbox Binary
Suite Teardown      Cleanup Test Artifacts

*** Variables ***
${BINARY}           ${CURDIR}/../thinbox
${TIMEOUT}          30s

*** Test Cases ***

TC-01 Binary Exists And Is Executable
    [Documentation]    Sanity check: thinbox binary is present after build.
    [Tags]             smoke
    File Should Exist    ${BINARY}
    ${result}=    Run Process    ${BINARY}    help
    ...           timeout=${TIMEOUT}    stderr=STDOUT
    Should Contain    ${result.stdout}    Usage

TC-02 Container Runs And Exits Cleanly
    [Documentation]    Run a simple echo inside a container and assert zero exit code.
    ...                Requires real Linux kernel — WSL2 blocks /proc/self/exe re-exec.
    [Tags]             smoke    lifecycle    requires-linux
    ${result}=    Run Thinbox Command    /bin/echo    hello thinbox
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain    ${result.stdout}    hello thinbox

TC-03 PID Namespace Isolation
    [Documentation]    The process should see itself as PID 1 (CLONE_NEWPID active).
    [Tags]             isolation    namespace    requires-linux
    ${result}=    Run Thinbox Command    /bin/sh    -c    echo $$
    Should Be Equal As Integers    ${result.rc}    0
    ${pid}=    Convert To Integer    ${result.stdout.strip()}
    Should Be True    ${pid} == 1
    ...    msg=Expected PID 1 inside container, got ${pid}

TC-04 UTS Namespace - Hostname Isolation
    [Documentation]    Container hostname must differ from host (CLONE_NEWUTS active).
    [Tags]             isolation    namespace    requires-linux
    ${host}=    Run Process    hostname    stdout=PIPE
    ${result}=    Run Thinbox Command    /bin/hostname
    Should Be Equal As Integers    ${result.rc}    0
    Should Not Be Equal    ${result.stdout.strip()}    ${host.stdout.strip()}
    ...    msg=Container hostname should be isolated from host

TC-05 Network Namespace Isolation
    [Documentation]    Only loopback should be visible inside container (CLONE_NEWNET active).
    [Tags]             isolation    namespace    requires-linux
    ${result}=    Run Thinbox Command    /bin/sh    -c    ip link show | grep -v lo | wc -l
    Should Be Equal As Integers    ${result.rc}    0
    Should Be Equal As Integers    ${result.stdout.strip()}    0
    ...    msg=Expected no non-loopback interfaces in isolated network namespace

TC-06 Fresh Proc Mount
    [Documentation]    /proc should show only container-owned PIDs (fresh mount active).
    [Tags]             isolation    filesystem    requires-linux
    ${result}=    Run Thinbox Command    /bin/sh    -c    ls /proc | grep -E '^[0-9]+$' | wc -l
    Should Be Equal As Integers    ${result.rc}    0
    ${count}=    Convert To Integer    ${result.stdout.strip()}
    Should Be True    ${count} < 5
    ...    msg=Fresh /proc should show only a handful of container PIDs, got ${count}

TC-07 Pivot Root Filesystem Isolation
    [Documentation]    Root filesystem must be Alpine, not the host OS.
    [Tags]             isolation    filesystem    requires-linux
    ${result}=    Run Thinbox Command    /bin/sh    -c    cat /etc/alpine-release
    Should Be Equal As Integers    ${result.rc}    0
    Should Not Be Empty    ${result.stdout.strip()}
    ...    msg=Expected Alpine rootfs inside container

TC-08 Non-Zero Exit Code Propagated
    [Documentation]    thinbox must propagate non-zero exit codes from the container.
    [Tags]             lifecycle    requires-linux
    ${result}=    Run Thinbox Command    /bin/sh    -c    exit 42
    Should Be Equal As Integers    ${result.rc}    42

TC-09 Invalid Command Returns Error
    [Documentation]    A non-existent binary inside the container should return non-zero.
    [Tags]             lifecycle    negative    requires-linux
    ${result}=    Run Thinbox Command    /nonexistent/binary
    Should Not Be Equal As Integers    ${result.rc}    0

TC-11 PS Shows Running Container
    [Documentation]    While a container is running, thinbox ps must show it with the correct image name.
    ...                A background sleep container is started, ps is polled, then the container is terminated.
    [Tags]             lifecycle    requires-linux
    ${handle}=    Run Thinbox Command Background    /bin/sleep    30
    Sleep    1s    reason=Allow state file to be written before querying ps
    ${result}=    Run Process    ${BINARY}    ps    timeout=${TIMEOUT}    stderr=STDOUT
    Terminate Process    ${handle}
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain    ${result.stdout}    alpine
    ...    msg=Expected running container with image 'alpine' in ps output

TC-10 Missing Subcommand Shows Usage
    [Documentation]    thinbox invoked with no arguments must fail with usage message.
    [Tags]             negative    smoke
    ${result}=    Run Process    ${BINARY}
    ...           timeout=${TIMEOUT}    stderr=STDOUT
    Should Not Be Equal As Integers    ${result.rc}    0
    Should Contain    ${result.stdout}    Usage
