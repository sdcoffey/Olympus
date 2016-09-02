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
const uglify = require('gulp-uglify');
const argv = require('yargs').argv;
const rename = require('gulp-rename');
const preprocess = require('gulp-preprocess');
const live_server = require('gulp-live-server');

var libs_prod = [
  'node_modules/angular2/bundles/angular2-polyfills.min.js',
  'node_modules/systemjs/dist/system.src.js',
  'node_modules/rxjs/bundles/Rx.min.js',
  'node_modules/angular2/bundles/angular2.min.js',
  'node_modules/angular2/bundles/http.min.js',
  'node_modules/angular2/bundles/router.min.js',
  'node_modules/es6-shim/es6-shim.min.js',
  'node_modules/systemjs/dist/system-polyfills.js',
  'node_modules/ng2-bootstrap/bundles/ng2.bootstrap.min.js',
];

var libs_dev = [
  'node_modules/angular2/bundles/angular2-polyfills.js',
  'node_modules/systemjs/dist/system.src.js',
  'node_modules/rxjs/bundles/Rx.js',
  'node_modules/angular2/bundles/angular2.dev.js',
  'node_modules/angular2/bundles/http.dev.js',
  'node_modules/angular2/bundles/router.dev.js',
  'node_modules/es6-shim/es6-shim.min.js',
  'node_modules/systemjs/dist/system-polyfills.js',
  'node_modules/ng2-bootstrap/bundles/ng2.bootstrap.js',
];

var production = !!argv.production;

gulp.task('clean', function () {
  return del.sync('dist/**/*');
});

gulp.task('browsersync', ['build'], function () {
  browsersync.init({
    server: {
      baseDir: 'dist'
    }
  });
});

gulp.task('compile', function () {
  var compiler = gulp.src('app/**/*.ts')
    .pipe(sourcemaps.init())
    .pipe(preprocess({ context: { PROD: production } }))
    .pipe(typescript(tscConfig.compilerOptions));

  if (production) {
    compiler.pipe(uglify());
  } else {
    compiler.pipe(sourcemaps.write('.'));
  }
  return compiler.pipe(gulp.dest('dist/app'));
});

gulp.task('sass', function () {
  var main = gulp.src('main.sass')
    .pipe(sass())
    .pipe(gulp.dest('dist'));

  var component = gulp.src('app/**/*.sass')
    .pipe(sass())
    .pipe(gulp.dest('dist/app'));

  return merge(main, component);
});

gulp.task('copy:libs', function () {
  return gulp.src(production ? libs_prod : libs_dev)
    .pipe(rename(function(path) {
      if (path.basename.indexOf("min") > 0 || path.basename.indexOf('dev') > 0) {
        path.basename = path.basename.split('.')[0];
      }
    }))
    .pipe(gulp.dest('dist/lib'));
});

gulp.task('copy:assets', function () {
  return gulp.src(['app/**/*.html', 'app/**/*.css', 'index.html', '!app/**/*.scss', '!app/**/*.ts'], {base: './'})
    .pipe(gulp.dest('dist'));
});

gulp.task('watch', ['build'], function () {
  livereload.listen();
  gulp.watch('app/**/*', ['build']);
});

gulp.task('serve', ['build'], function() {
  var server = live_server.static('dist', '3001');
  server.start();
  gulp.watch('app/**/*', ['build'], function (file) {
    server.notify.apply(server, [file]);
  })
});

gulp.task('build', ['clean', 'compile', 'sass', 'copy:libs', 'copy:assets']);
gulp.task('default', ['build']);
