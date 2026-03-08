/*
 Navicat Premium Dump SQL

 Source Server         : mysql-on-vmware-Ubuntu
 Source Server Type    : MySQL
 Source Server Version : 80044 (8.0.44)
 Source Host           : 10.201.144.200:3306
 Source Schema         : video_server

 Target Server Type    : MySQL
 Target Server Version : 80044 (8.0.44)
 File Encoding         : 65001

 Date: 22/01/2026 20:59:04
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for comments
-- ----------------------------
DROP TABLE IF EXISTS `comments`;
CREATE TABLE `comments`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `comment_id` char(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `video_id` char(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `author_id` bigint NOT NULL,
  `content` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `create_time` datetime NOT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of comments
-- ----------------------------
INSERT INTO `comments` VALUES (1, 'e3f4dbe6-430e-45c0-9f36-c865880cb533', 'dd8f27e3-0fed-426b-b827-02920c985e4a', 1, 'this is a test comment', '2025-12-28 18:48:52');

-- ----------------------------
-- Table structure for sessions
-- ----------------------------
DROP TABLE IF EXISTS `sessions`;
CREATE TABLE `sessions`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `session_id` char(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `user_id` bigint NOT NULL,
  `user_name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL,
  `ttl` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of sessions
-- ----------------------------
INSERT INTO `sessions` VALUES (1, '903c854b-a110-4f6d-a901-e4ff9431714b', 2, 'testuser', '1767429495');

-- ----------------------------
-- Table structure for users
-- ----------------------------
DROP TABLE IF EXISTS `users`;
CREATE TABLE `users`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `password` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `isVaild` int NOT NULL DEFAULT 1 COMMENT '用户状态，是否合法',
  `creatAt` datetime NOT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 3 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of users
-- ----------------------------
INSERT INTO `users` VALUES (1, 'yanghao', '123456', 1, '2025-12-28 18:48:52');
INSERT INTO `users` VALUES (2, 'testuser', '123456', 1, '2025-12-28 23:23:34');

-- ----------------------------
-- Table structure for video_delete_record
-- ----------------------------
DROP TABLE IF EXISTS `video_delete_record`;
CREATE TABLE `video_delete_record`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `vid` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci ROW_FORMAT = Dynamic;

-- ----------------------------
-- Records of video_delete_record
-- ----------------------------

-- ----------------------------
-- Table structure for video_info
-- ----------------------------
DROP TABLE IF EXISTS `video_info`;
CREATE TABLE `video_info`  (
  `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT,
  `vid` char(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `author_id` bigint NOT NULL,
  `name` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `create_time` datetime NOT NULL,
  `click_count` bigint(20) UNSIGNED ZEROFILL NULL DEFAULT NULL COMMENT '播放次数',
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 2 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = DYNAMIC;

-- ----------------------------
-- Records of video_info
-- ----------------------------
INSERT INTO `video_info` VALUES (1, 'dd8f27e3-0fed-426b-b827-02920c985e4a', 1, 'test_video', '2025-12-28 18:48:52', NULL);

SET FOREIGN_KEY_CHECKS = 1;
