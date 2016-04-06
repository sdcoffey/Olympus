const tscConfig = require('./tsconfig.json');
const gulp = require('gulp');
const del = require('del');
const typescript = require('gulp-typescript');
const sourcemaps = require('gulp-sourcemaps');
const watch = require('gulp-watch');
const sass = require('gulp-sass');
const debug = require('gulp-debug');
const merge = require('merge-stream');
const livereload = require('gulp-livereload');
const browsersync = require('browser-sync').create();

gulp.task('clean', function() {
  return del.sync('dist/**/*');
});

gulp.task('browsersync', ['build'], function() {
    browsersync.init({
      server: {
          baseDir: 'dist'
      }
    });
});

gulp.task('compile', function() {
  return gulp
    .src('app/**/*.ts')
    .pipe(sourcemaps.init())
    .pipe(typescript(tscConfig.compilerOptions))
    .pipe(sourcemaps.write('.'))
    .pipe(gulp.dest('dist/app'));
});

gulp.task('sass', function() {
  var main = gulp.src('main.sass')
    .pipe(sass())
    .pipe(gulp.dest('dist'));

  var component = gulp.src('app/**/*.sass')
    .pipe(sass())
    .pipe(gulp.dest('dist/app'))

  return merge(main, component);
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
  return gulp.src(['app/**/*.html', 'app/**/*.css', 'index.html', '!app/**/*.scss', '!app/**/*.ts'], {base: './'})
    .pipe(gulp.dest('dist'));
});

gulp.task('watch', ['build'],  function() {
  livereload.listen();
  gulp.watch('app/**/*', ['build']);
});

gulp.task('build', ['clean', 'compile', 'sass', 'copy:libs', 'copy:assets']);
gulp.task('default', ['build']);
