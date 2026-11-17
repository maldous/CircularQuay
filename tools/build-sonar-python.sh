#!/usr/bin/env bash
# Build only the selected upstream community analyzer in task-owned CI state.
# The JAR stays outside uploaded evidence; keep upstream tests and notices intact.
set -euo pipefail
umask 077
# Emit only fixed stage labels and exit codes, never commands, paths or environment.
analyzer_stage=arguments
analyzer_preflight_stage() {
  analyzer_stage=$1
  printf 'sonar-analyzer-stage:%s\n' "$analyzer_stage" >&2 || :
}
trap 'analyzer_status=$?; printf "sonar-analyzer-error:stage=%s,status=%d\n" "$analyzer_stage" "$analyzer_status" >&2 || :; exit "$analyzer_status"' ERR
analyzer_preflight_stage arguments
test "$#" = 3 || { printf 'sonar-analyzer-error:stage=arguments,status=2\n' >&2 || :; exit 2; }
build_root=$1
evidence=$2
python_executable=$3
analyzer_preflight_stage executable-preflight
test -n "${JAVA_HOME:-}" && test -x "$python_executable"
analyzer_preflight_stage private-root-preflight
test ! -e "$build_root"
# Maven launcher option splitting must not reinterpret a path containing whitespace.
[[ "$build_root" =~ ^/[A-Za-z0-9_./-]+$ ]]
analyzer_preflight_stage state-directories
mkdir -p "$build_root" "$evidence"
mkdir "$build_root/bin" "$build_root/home" "$build_root/tmp" "$build_root/repository"
analyzer_preflight_stage interpreter-links
ln -s "$python_executable" "$build_root/bin/python"
ln -s "$python_executable" "$build_root/bin/python3"
build_path="$build_root/bin:$JAVA_HOME/bin:/usr/bin:/bin"
# Pin the empty user settings and Java paths. Global settings remain the
# checksum-verified Maven distribution defaults, including its HTTP blocker.
analyzer_preflight_stage maven-settings
cat >"$build_root/settings.xml" <<'SETTINGS'
<settings xmlns="http://maven.apache.org/SETTINGS/1.2.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.2.0 https://maven.apache.org/xsd/settings-1.2.0.xsd"/>
SETTINGS
analyzer_preflight_stage interpreter-version
# The original test uses stdout even if the probe exits nonzero. Keep that
# predicate, while withholding arbitrary probe diagnostics and non-version text.
analyzer_python_version="$("$python_executable" -c 'import platform; print(platform.python_version())' 2>/dev/null)" || :
if [[ "$analyzer_python_version" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
  printf 'sonar-analyzer-python-version:%s\n' "$analyzer_python_version" >&2 || :
else
  printf 'sonar-analyzer-python-version:noncanonical\n' >&2 || :
fi
test "$analyzer_python_version" = 3.14.3
analyzer_preflight_stage timing-tool
test -x /usr/bin/time
archive="$build_root/maven.tar.gz"
analyzer_preflight_stage maven-download
curl --connect-timeout 15 --max-time 180 --max-filesize 20000000 --fail --silent --show-error --location \
  https://archive.apache.org/dist/maven/maven-3/3.9.16/binaries/apache-maven-3.9.16-bin.tar.gz \
  --output "$archive"
analyzer_preflight_stage maven-archive-digest
printf '%s  %s\n' 80ffca22aed9e8b9713a232f3394fd81d7f20322df75efdb2b047dbd3e3a23bb "$archive" | sha256sum -c -
analyzer_preflight_stage maven-extraction
tar -xzf "$archive" -C "$build_root"
maven="$build_root/apache-maven-3.9.16/bin/mvn"
global_settings="$build_root/apache-maven-3.9.16/conf/settings.xml"
analyzer_preflight_stage maven-launcher-and-http-blocker
test -x "$maven"
test -s "$global_settings"
grep -Fq '<id>maven-default-http-blocker</id>' "$global_settings"
grep -Fq '<mirrorOf>external:http:*</mirrorOf>' "$global_settings"
grep -Fq '<blocked>true</blocked>' "$global_settings"
source_root="$build_root/source"
analyzer_preflight_stage upstream-initialization
git init -q "$source_root"
git -C "$source_root" remote add origin https://github.com/SonarSource/sonar-python.git
analyzer_preflight_stage upstream-fetch
timeout 120s git -c credential.helper= -C "$source_root" fetch --no-tags --depth=1 origin 2c1b711f17649326a2a66651a87d1136e728f403
analyzer_preflight_stage upstream-checkout
git -C "$source_root" checkout --detach -q FETCH_HEAD
analyzer_preflight_stage upstream-identity-and-license
test "$(git -C "$source_root" rev-parse HEAD)" = 2c1b711f17649326a2a66651a87d1136e728f403
test "$(git -C "$source_root" rev-parse 'HEAD^{tree}')" = dbf8eabc4ff128712c66eed25b19ea2805148a18
git -C "$source_root" diff --quiet
git -C "$source_root" diff --cached --quiet
test -s "$source_root/LICENSE.txt"
analyzer_preflight_stage build-input-record
{
  printf 'upstream_commit=2c1b711f17649326a2a66651a87d1136e728f403\nupstream_tree=dbf8eabc4ff128712c66eed25b19ea2805148a18\n'
  printf 'maven_archive_sha256=80ffca22aed9e8b9713a232f3394fd81d7f20322df75efdb2b047dbd3e3a23bb\n'
  "$JAVA_HOME/bin/java" -version 2>&1
  env -i PATH="$build_path" JAVA_HOME="$JAVA_HOME" HOME="$build_root/home" TMPDIR="$build_root/tmp" MAVEN_SKIP_RC=true MAVEN_OPTS="-Duser.home=$build_root/home -Djava.io.tmpdir=$build_root/tmp" "$maven" -version
  "$python_executable" --version
} >"$evidence/analyzer-build-inputs.txt"
cp "$source_root/LICENSE.txt" "$evidence/analyzer-LICENSE.txt"
analyzer_preflight_stage upstream-maven-build
(
  cd "$source_root"
  # Same upstream recipe as the accepted native build; no test/license skips.
  env -i PATH="$build_path" JAVA_HOME="$JAVA_HOME" HOME="$build_root/home" TMPDIR="$build_root/tmp" LANG=C.UTF-8 IS_COMMUNITY=true MAVEN_SKIP_RC=true MAVEN_OPTS="-Duser.home=$build_root/home -Djava.io.tmpdir=$build_root/tmp" \
    timeout --kill-after=15s 600s /usr/bin/time -v -o "$evidence/analyzer-build-resources.txt" \
    "$maven" -B -ntp -P=-private --settings "$build_root/settings.xml" --global-settings "$global_settings" -Dmaven.repo.local="$build_root/repository" clean install \
    >"$evidence/analyzer-build.log" 2>&1
)
analyzer_preflight_stage post-build-clean-source
git -C "$source_root" diff --quiet
git -C "$source_root" diff --cached --quiet
jar="$source_root/sonar-python-plugin/target/sonar-python-plugin-5.33-SNAPSHOT.jar"
analyzer_preflight_stage analyzer-artifact
test -s "$jar"
cp "$jar" "$build_root/analyzer.jar"
chmod 444 "$build_root/analyzer.jar"
sha256sum "$build_root/analyzer.jar" >"$evidence/analyzer-jar.sha256"
unzip -p "$build_root/analyzer.jar" META-INF/MANIFEST.MF >"$evidence/analyzer-manifest.txt"
analyzer_preflight_stage analyzer-manifest-contract
grep -q '^Plugin-Key: python' "$evidence/analyzer-manifest.txt"
grep -q '^Plugin-Version: 5.33-SNAPSHOT' "$evidence/analyzer-manifest.txt"
# No old-hash equality assertion: build date/time are known variable inputs.
analyzer_preflight_stage complete
