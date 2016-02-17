var files = ['app/**'];

module.exports = function(grunt) {
  grunt.initConfig({
    watch: {
      scripts: {
        files: ['app/**'],
        tasks: ['copy']
      },
    },
    copy: {
      main: {
        files: [
          {expand: true, cwd: 'app/', src: ['**'], dest: '/usr/local/etc/olympus/www'},
        ],
      },
    },
  });

  grunt.loadNpmTasks('grunt-contrib-watch');
  grunt.loadNpmTasks('grunt-contrib-copy');
  grunt.registerTask('default', ['watch']);
}
