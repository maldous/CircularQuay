#!/usr/bin/env bash
# Build only the selected upstream community analyzer in task-owned CI state.
# The JAR stays outside uploaded evidence; keep upstream tests and notices intact.
set -euo pipefail
umask 077
test "$#" = 3 || exit 2
build_root=$1
evidence=$2
python_executable=$3
test -n "${JAVA_HOME:-}" && test -x "$python_executable"
test ! -e "$build_root"
# Maven launcher option splitting must not reinterpret a path containing whitespace.
[[ "$build_root" =~ ^/[A-Za-z0-9_./-]+$ ]]
mkdir -p "$build_root" "$evidence"
mkdir "$build_root/bin" "$build_root/home" "$build_root/tmp" "$build_root/repository"
ln -s "$python_executable" "$build_root/bin/python"
ln -s "$python_executable" "$build_root/bin/python3"
build_path="$build_root/bin:$JAVA_HOME/bin:/usr/bin:/bin"
# Pin the empty user settings and Java paths. Global settings remain the
# checksum-verified Maven distribution defaults, including its HTTP blocker.
cat >"$build_root/settings.xml" <<'SETTINGS'
<settings xmlns="http://maven.apache.org/SETTINGS/1.2.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.2.0 https://maven.apache.org/xsd/settings-1.2.0.xsd"/>
SETTINGS
test "$("$python_executable" -c 'import platform; print(platform.python_version())')" = 3.14.3
test -x /usr/bin/time
archive="$build_root/maven.tar.gz"
curl --connect-timeout 15 --max-time 180 --max-filesize 20000000 --fail --silent --show-error --location \
  https://archive.apache.org/dist/maven/maven-3/3.9.16/binaries/apache-maven-3.9.16-bin.tar.gz \
  --output "$archive"
printf '%s  %s\n' 80ffca22aed9e8b9713a232f3394fd81d7f20322df75efdb2b047dbd3e3a23bb "$archive" | sha256sum -c -
tar -xzf "$archive" -C "$build_root"
maven="$build_root/apache-maven-3.9.16/bin/mvn"
global_settings="$build_root/apache-maven-3.9.16/conf/settings.xml"
test -x "$maven"
test -s "$global_settings"
grep -Fq '<id>maven-default-http-blocker</id>' "$global_settings"
grep -Fq '<mirrorOf>external:http:*</mirrorOf>' "$global_settings"
grep -Fq '<blocked>true</blocked>' "$global_settings"
source_root="$build_root/source"
git init -q "$source_root"
git -C "$source_root" remote add origin https://github.com/SonarSource/sonar-python.git
timeout 120s git -c credential.helper= -C "$source_root" fetch --no-tags --depth=1 origin 2c1b711f17649326a2a66651a87d1136e728f403
git -C "$source_root" checkout --detach -q FETCH_HEAD
test "$(git -C "$source_root" rev-parse HEAD)" = 2c1b711f17649326a2a66651a87d1136e728f403
test "$(git -C "$source_root" rev-parse 'HEAD^{tree}')" = dbf8eabc4ff128712c66eed25b19ea2805148a18
git -C "$source_root" diff --quiet
git -C "$source_root" diff --cached --quiet
test -s "$source_root/LICENSE.txt"
{
  printf 'upstream_commit=2c1b711f17649326a2a66651a87d1136e728f403\nupstream_tree=dbf8eabc4ff128712c66eed25b19ea2805148a18\n'
  printf 'maven_archive_sha256=80ffca22aed9e8b9713a232f3394fd81d7f20322df75efdb2b047dbd3e3a23bb\n'
  "$JAVA_HOME/bin/java" -version 2>&1
  env -i PATH="$build_path" JAVA_HOME="$JAVA_HOME" HOME="$build_root/home" TMPDIR="$build_root/tmp" MAVEN_SKIP_RC=true MAVEN_OPTS="-Duser.home=$build_root/home -Djava.io.tmpdir=$build_root/tmp" "$maven" -version
  "$python_executable" --version
} >"$evidence/analyzer-build-inputs.txt"
cp "$source_root/LICENSE.txt" "$evidence/analyzer-LICENSE.txt"
(
  cd "$source_root"
  # Same upstream recipe as the accepted native build; no test/license skips.
  env -i PATH="$build_path" JAVA_HOME="$JAVA_HOME" HOME="$build_root/home" TMPDIR="$build_root/tmp" LANG=C.UTF-8 IS_COMMUNITY=true MAVEN_SKIP_RC=true MAVEN_OPTS="-Duser.home=$build_root/home -Djava.io.tmpdir=$build_root/tmp" \
    timeout --kill-after=15s 600s /usr/bin/time -v -o "$evidence/analyzer-build-resources.txt" \
    "$maven" -B -ntp -P=-private --settings "$build_root/settings.xml" --global-settings "$global_settings" -Dmaven.repo.local="$build_root/repository" clean install \
    >"$evidence/analyzer-build.log" 2>&1
)
git -C "$source_root" diff --quiet
git -C "$source_root" diff --cached --quiet
jar="$source_root/sonar-python-plugin/target/sonar-python-plugin-5.33-SNAPSHOT.jar"
test -s "$jar"
cp "$jar" "$build_root/analyzer.jar"
chmod 444 "$build_root/analyzer.jar"
sha256sum "$build_root/analyzer.jar" >"$evidence/analyzer-jar.sha256"
unzip -p "$build_root/analyzer.jar" META-INF/MANIFEST.MF >"$evidence/analyzer-manifest.txt"
grep -q '^Plugin-Key: python' "$evidence/analyzer-manifest.txt"
grep -q '^Plugin-Version: 5.33-SNAPSHOT' "$evidence/analyzer-manifest.txt"
# No old-hash equality assertion: build date/time are known variable inputs.
