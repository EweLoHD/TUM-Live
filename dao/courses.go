package dao

import (
	"context"
	"errors"
	"fmt"
	"github.com/joschahenningsen/TUM-Live/model"
	"time"

	"github.com/RBG-TUM/commons"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

//GetCurrentOrNextLectureForCourse Gets the next lecture for a course or the lecture that is currently live. Error otherwise.
func GetCurrentOrNextLectureForCourse(ctx context.Context, courseID uint) (model.Stream, error) {
	var res model.Stream
	err := DB.Model(&model.Stream{}).Preload("Chats").Order("start").First(&res, "course_id = ? AND (end > NOW() OR live_now)", courseID).Error
	return res, err
}

// GetAllCourses retrieves all courses from the database
func GetAllCourses() ([]model.Course, error) {
	cachedCourses, found := Cache.Get("allCourses")
	if found {
		return cachedCourses.([]model.Course), nil
	}
	var courses []model.Course
	err := DB.Preload("Streams").Find(&courses).Error
	if err == nil {
		Cache.SetWithTTL("allCourses", courses, 1, time.Minute)
	}
	return courses, err
}

func GetCourseForLecturerIdByYearAndTerm(c context.Context, year int, term string, userId uint) ([]model.Course, error) {
	var res []model.Course
	err := DB.Model(&model.Course{}).Find(&res, "user_id = ? AND year = ? AND teaching_term = ?", userId, year, term).Error
	return res, err
}

func GetAdministeredCoursesByUserId(ctx context.Context, userid uint) (courses []model.Course, err error) {
	cachedCourses, found := Cache.Get(fmt.Sprintf("coursesByUserID%v", userid))
	if found {
		return cachedCourses.([]model.Course), nil
	}
	isAdmin, err := IsUserAdmin(ctx, userid)
	if err != nil {
		return nil, err
	}
	var foundCourses []model.Course
	// all courses for admins
	if isAdmin {
		dbErr := DB.Preload("Streams", func(db *gorm.DB) *gorm.DB {
			return db.Order("start asc")
		}).Find(&foundCourses).Error
		if dbErr == nil {
			Cache.SetWithTTL(fmt.Sprintf("coursesByUserID%v", userid), foundCourses, 1, time.Minute)
		}
		return foundCourses, dbErr
	}
	dbErr := DB.Preload("Streams", func(db *gorm.DB) *gorm.DB {
		return db.Order("start asc")
	}).Find(&foundCourses, "user_id = ?", userid).Error

	if err != nil && errors.Is(dbErr, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var administeredCourses []model.Course
	err = DB.Raw("SELECT * FROM courses JOIN course_admins ON courses.id = course_admins.course_id WHERE course_admins.user_id = ? AND courses.deleted_at IS NULL", userid).Scan(&administeredCourses).Error
	if err != nil {
		return nil, err
	}
	foundCourses = append(foundCourses, administeredCourses...)
	foundCourses = commons.Unique(foundCourses, func(c model.Course) uint { return c.ID })
	if dbErr == nil {
		Cache.SetWithTTL(fmt.Sprintf("coursesByUserID%v", userid), foundCourses, 1, time.Minute)
	}
	return foundCourses, dbErr
}

func GetAllCoursesForSemester(year int, term string, ctx context.Context) (courses []model.Course) {
	span := sentry.StartSpan(ctx, "SQL: GetAllCoursesForSemester")
	defer span.Finish()
	var foundCourses []model.Course
	DB.Preload("Streams", func(db *gorm.DB) *gorm.DB {
		return db.Order("start asc")
	}).Find(&foundCourses, "teaching_term = ? AND year = ?", term, year)
	return foundCourses
}

func GetPublicCourses(year int, term string) (courses []model.Course, err error) {
	cachedCourses, found := Cache.Get(fmt.Sprintf("publicCourses%d%v", year, term))
	if found {
		return cachedCourses.([]model.Course), err
	}
	var publicCourses []model.Course

	err = DB.Preload("Streams", func(db *gorm.DB) *gorm.DB {
		return db.Order("start asc")
	}).Find(&publicCourses, "visibility = 'public' AND teaching_term = ? AND year = ?",
		term, year).Error

	if err == nil {
		Cache.SetWithTTL(fmt.Sprintf("publicCourses%d%v", year, term), publicCourses, 1, time.Minute)
	}
	return publicCourses, err
}

func GetPublicAndLoggedInCourses(year int, term string) (courses []model.Course, err error) {
	cachedCourses, found := Cache.Get(fmt.Sprintf("publicAndLoggedInCourses%d%v", year, term))
	if found {
		return cachedCourses.([]model.Course), err
	}
	var publicCourses []model.Course

	err = DB.Preload("Streams", func(db *gorm.DB) *gorm.DB {
		return db.Order("start asc")
	}).Find(&publicCourses,
		"(visibility = 'public' OR visibility = 'loggedin') AND teaching_term = ? AND year = ?", term, year).Error
	if err == nil {
		Cache.SetWithTTL(fmt.Sprintf("publicAndLoggedInCourses%d%v", year, term), publicCourses, 1, time.Minute)
	}
	return publicCourses, err
}

func DeleteCourse(course model.Course) {
	for _, stream := range course.Streams {
		err := DB.Delete(&stream).Error
		if err != nil {
			log.WithError(err).Error("Can't delete stream")
		}
	}
	err := DB.Model(&course).Updates(map[string]interface{}{"live_enabled": false, "vod_enabled": false}).Error
	if err != nil {
		log.WithError(err).Error("Can't update course settings when deleting")
	}
	err = DB.Delete(&course).Error
	if err != nil {
		log.WithError(err).Error("Can't delete course")
	}
}

func GetCourseByToken(token string) (course model.Course, err error) {
	err = DB.Unscoped().
		Preload("Streams", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		First(&course, "token = ?", token).Error
	return
}

func GetCourseById(ctx context.Context, id uint) (course model.Course, err error) {
	var foundCourse model.Course
	dbErr := DB.Preload("Streams.Stats").Preload("Streams.Files").Preload("Streams", func(db *gorm.DB) *gorm.DB {
		return db.Order("streams.start desc")
	}).Find(&foundCourse, "id = ?", id).Error
	return foundCourse, dbErr
}

func GetInvitedUsersForCourse(course *model.Course) error {
	return DB.Preload("Users", "role = ?", model.GenericType).Find(course).Error
}

func GetCourseBySlugYearAndTerm(ctx context.Context, slug string, term string, year int) (model.Course, error) {
	cachedCourses, found := Cache.Get(fmt.Sprintf("courseBySlugYearAndTerm%v%v%v", slug, term, year))
	if found {
		return cachedCourses.(model.Course), nil
	}
	var course model.Course
	err := DB.Preload("Streams.Units", func(db *gorm.DB) *gorm.DB {
		return db.Order("unit_start desc")
	}).Preload("Streams", func(db *gorm.DB) *gorm.DB {
		return db.Order("start desc")
	}).Where("teaching_term = ? AND slug = ? AND year = ?", term, slug, year).First(&course).Error
	if err == nil {
		Cache.SetWithTTL(fmt.Sprintf("courseBySlugYearAndTerm%v%v%v", slug, term, year), course, 1, time.Minute)
	}
	return course, err
}

func GetAllCoursesWithTUMIDForSemester(ctx context.Context, year int, term string) (courses []model.Course, err error) {
	var foundCourses []model.Course
	dbErr := DB.Where("tum_online_identifier <> '' AND year = ? AND teaching_term = ?", year, term).Find(&foundCourses).Error
	return foundCourses, dbErr
}

func UpdateCourseMetadata(ctx context.Context, course model.Course) {
	defer Cache.Clear()
	DB.Save(&course)
}

func UnDeleteCourse(ctx context.Context, course model.Course) error {
	return DB.Exec("UPDATE courses SET deleted_at = NULL WHERE id = ?", course.ID).Error
}

func UpdateCourse(ctx context.Context, course model.Course) error {
	defer Cache.Clear()
	return DB.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&course).Error
}

// CreateCourse creates a new course, if keep is false, deleted_at is set to NOW(),
// letting the user manually create the course again (opt-in)
func CreateCourse(ctx context.Context, course *model.Course, keep bool) error {
	defer Cache.Clear()
	err := DB.Create(&course).Error
	if err != nil {
		return err
	}
	if !keep {
		err = DB.Model(&course).Updates(map[string]interface{}{"live_enabled": "0"}).Error
		if err != nil {
			log.WithError(err).Error("Can't update live enabled state")
		}
		return DB.Delete(&course).Error
	}
	return nil
}

func GetAvailableSemesters(c context.Context) []Semester {
	if cached, found := Cache.Get("getAllSemesters"); found {
		return cached.([]Semester)
	} else {
		var semesters []Semester
		DB.Raw("SELECT year, teaching_term from courses " +
			"group by year, teaching_term " +
			"order by year desc, teaching_term desc").Scan(&semesters)
		Cache.SetWithTTL("getAllSemesters", semesters, 1, time.Hour)
		return semesters
	}
}

// GetCourseByShortLink returns the course associated with the given short link (e.g. EIDI2022)
func GetCourseByShortLink(link string) (model.Course, error) {
	var sl model.ShortLink
	err := DB.First(&sl, "link = ?", link).Error
	if err != nil {
		return model.Course{}, err
	}
	course, err := GetCourseById(context.Background(), sl.CourseId)
	return course, err
}

// GetCourseAdmins returns the admins of the given course excluding the creator (usually system) and the tumlive admins
func GetCourseAdmins(courseID uint) ([]model.User, error) {
	var admins []model.User
	err := DB.Raw("select u.* from courses "+
		"join course_admins ca on courses.id = ca.course_id "+
		"join users u on u.id = ca.user_id "+
		"where course_id = ?", courseID).
		Scan(&admins).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return admins, nil
	}
	return admins, err
}

func AddAdminToCourse(userID uint, courseID uint) error {
	defer Cache.Clear()
	return DB.Exec("insert into course_admins (user_id, course_id) values (?, ?)", userID, courseID).Error
}

func RemoveAdminFromCourse(userID uint, courseID uint) error {
	defer Cache.Clear()
	return DB.Exec("delete from course_admins where user_id = ? and course_id = ?", userID, courseID).Error
}

type Semester struct {
	TeachingTerm string
	Year         int
}
