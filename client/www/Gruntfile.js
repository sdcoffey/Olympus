var files = ['index.html', 'template/*.html', 'style/*.css'];

module.exports = function(grunt) {
  grunt.initConfig({
    watch: {
      scripts: {
        files: files,
        tasks: ['copy']
      },
    },
    copy: {
      main: {
        files: [
          {expand: true, src: files, dest: '/usr/local/etc/olympus/www'},
        ],
      },
    },
  });

  grunt.loadNpmTasks('grunt-contrib-watch');
  grunt.loadNpmTasks('grunt-contrib-copy');
  grunt.registerTask('default', ['watch']);
}
