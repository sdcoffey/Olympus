const gulp = require('gulp');
const del = require('del');
const typescript = require('gulp-typescript');
const sourcemaps = require('gulp-sourcemaps');
const watch = require('gulp-watch');
const scss = require('gulp-scss');
const tscConfig = require('./tsconfig.json');

gulp.task('clean', function() {
  return del.sync('dist/**/*');
});

gulp.task('compile', function() {
  return gulp
    .src('app/**/*.ts')
    .pipe(sourcemaps.init())
    .pipe(typescript(tscConfig.compilerOptions))
    .pipe(sourcemaps.write('.'))
    .pipe(gulp.dest('dist/app'));
});

gulp.task('compile-scss', function() {
  return gulp.src('/app/**/*.scss')
    .pipe(scss({"bundleExec":true}))
    .pipe(gulp.dest('dist/app'));
});

gulp.task('copy:libs', function() {
  return gulp.src([
    'node_modules/angular2/bundles/angular2-polyfills.js',
    'node_modules/systemjs/dist/system.src.js',
    'node_modules/rxjs/bundles/Rx.js',
    'node_modules/angular2/bundles/angular2.dev.js',
    'node_modules/angular2/bundles/http.dev.js',
    'node_modules/angular2/bundles/router.dev.js',
    'node_modules/es6-shim/es6-shim.min.js',
    'node_modules/systemjs/dist/system-polyfills.js',
    'node_modules/angular2/es6/dev/src/testing/shims_for_IE.js'
  ])
  .pipe(gulp.dest('dist/lib'));
});

gulp.task('copy:assets', function() {
  return gulp.src(['app/**/*', 'index.html', '!app/**/*.ts'], {base: './'})
    .pipe(gulp.dest('dist'));
});

gulp.task('watch', ['build'],  function() {
  gulp.watch('app/**/*', ['build']);
});

gulp.task('build', ['clean', 'compile', 'compile-scss', 'copy:libs', 'copy:assets']);
gulp.task('default', ['build']);
